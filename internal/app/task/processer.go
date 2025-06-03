package task

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

const MaxBitrateKbps = 5000 // Максимальный битрейт в кбит/с
const AVC = "libx264"       // Кодек для видео
const AAC = "aac"           // Кодек для аудио

const (
	MastePLName            = "master.m3u8"
	VariantPlaylistPattern = "stream_%v.m3u8"   // Шаблон для плейлистов HLS
	SegmentPattern         = "segment_%v_%d.ts" // Шаблон для сегментов HLS

)

type Processer interface {
	Process(t VideoTask, videoURL string, outputDir string) error
}

type VideoProcess struct {
}

func NewVideoProcess() *VideoProcess {
	return &VideoProcess{}
}

type Quality struct {
	Name        string
	Width       int
	Height      int
	BitrateKbps int
}

// Processer реализует интерфейс Processer и отвечает за обработку видео.
// Он принимает VideoTask, URL видео и директорию для сохранения обработанного видео.
// Внутри он проверяет и генерирует доступные качества, а затем создает HLS-плейлисты и сегменты.
func (vh *VideoProcess) Process(t VideoTask, videoURL string, outputDir string) error {
	// Получаем доступные качества видео
	q, err := vh.checkAndGenerateQualities(videoURL)
	if err != nil {
		log.Println("Ошибка при проверке и генерации качеств:", err)
		return fmt.Errorf("ошибка при проверке и генерации качеств: %w", err)
	}

	err = vh.generateHLS(videoURL, outputDir, q)
	if err != nil {
		log.Println("Ошибка при генерации HLS:", err)
		return fmt.Errorf("ошибка при генерации HLS: %w", err)
	}
	log.Printf("Видео %s успешно обработано и сохранено в директорию %s с качествами: %+v\n", t.VideoID, outputDir, q)
	return nil
}

// checkAndGenerateQualities проверяет метаданные видео и генерирует доступные качества.
// Если не удается получить метаданные или сгенерировать качества, возвращает ошибку.
func (vh *VideoProcess) checkAndGenerateQualities(videoURL string) ([]Quality, error) {
	// Получаем метаданные видео
	meta, err := vh.getVideoMetadata(videoURL)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить метаданные видео: %w", err)
	}
	log.Printf("Метаданные видео: %+v\n", meta)

	// Генерируем доступные качества на основе метаданных
	qualities := vh.autoConfig(meta)
	if len(qualities) == 0 {
		return nil, fmt.Errorf("не удалось сгенерировать доступные качества для видео %s", videoURL)
	}

	log.Printf("Сгенерированные качества: %+v\n", qualities)

	return qualities, nil
}

type VideoMetadata struct {
	Width         int
	Height        int
	SourceBitrate float64 // в кбит/с
}

type probeMetadata struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width,omitempty"`
		Height    int    `json:"height,omitempty"`
		BitRate   string `json:"bit_rate,omitempty"`
	} `json:"streams"`
	Format struct {
		BitRate string `json:"bit_rate,omitempty"`
	} `json:"format"`
}

// getVideoMetadata получает метаданные видео с помощью ffprobe и возвращает структуру VideoMetadata.
func (vh *VideoProcess) getVideoMetadata(videoURL string) (VideoMetadata, error) {
	// 1. Вызываем ffprobe для получения метаданных видео.
	rawJSON, err := ffmpeg_go.Probe(videoURL)
	if err != nil {
		return VideoMetadata{}, fmt.Errorf("не удалось вызвать ffprobe: %w", err)
	}

	// 2. Парсим полученный JSON в структуру probeMetadata.
	var meta probeMetadata
	if err := json.Unmarshal([]byte(rawJSON), &meta); err != nil {
		return VideoMetadata{}, fmt.Errorf("ошибка парсинга JSON-ответа от ffprobe: %w", err)
	}

	// 3. Находим первый видеопоток (codec_type == "video").
	//    Если ни одного «video» в streams нет — возвращаем ошибку.
	var vidStream struct {
		Width   int
		Height  int
		BitRate string
	}
	found := false
	for _, s := range meta.Streams {
		if s.CodecType == "video" {
			vidStream.Width = s.Width
			vidStream.Height = s.Height
			vidStream.BitRate = s.BitRate
			found = true
			break
		}
	}
	if !found {
		return VideoMetadata{}, fmt.Errorf("не найден видеопоток в файле %s", videoURL)
	}

	// 4. Получаем битрейт видео.
	//	  Сначала пробуем взять из meta.Format.BitRate, если там пусто — берем из видеопотока.
	rawBitrate := meta.Format.BitRate
	if rawBitrate == "" {
		rawBitrate = vidStream.BitRate
	}
	if rawBitrate == "" {
		return VideoMetadata{
			Width:         vidStream.Width,
			Height:        vidStream.Height,
			SourceBitrate: 0,
		}, nil
	}

	// 5. Конвертируем строку "битрейт"  в int.
	totalBitrate, err := strconv.Atoi(rawBitrate)
	if err != nil {
		return VideoMetadata{
				Width:         vidStream.Width,
				Height:        vidStream.Height,
				SourceBitrate: 0,
			},
			fmt.Errorf("ошибка преобразования bitrate (%q) в int: %w", rawBitrate, err)
	}

	return VideoMetadata{
		Width:         vidStream.Width,
		Height:        vidStream.Height,
		SourceBitrate: float64(totalBitrate) / 1000, // переводим в кбит/с
	}, nil
}

func (vh *VideoProcess) autoConfig(meta VideoMetadata) []Quality {
	maxHeightVideo := meta.Height // Не превышаем исходное

	// Рассчитываем битрейт для максимального качества
	var baseRate float64 = (float64(meta.Width*meta.Height*30) * 0.2) / 1000 // кбит/с
	if baseRate > meta.SourceBitrate {
		baseRate = meta.SourceBitrate * 0.9 // Не превышаем исходный
	}

	// Генерируем профили
	var profiles []Quality
	for _, res := range []struct {
		w    int
		h    int
		name string
	}{
		{1920, 1080, "1080p"}, {1280, 720, "720p"}, {854, 480, "480p"}, {640, 360, "360p"}, // TODO: тут можно и битрейт сделать динамическим
	} {
		if res.h > maxHeightVideo {
			continue
		}
		rate := baseRate * (float64(res.h) / float64(maxHeightVideo)) * 0.8
		if rate > MaxBitrateKbps {
			rate = MaxBitrateKbps // Ограничиваем битрейт
		}
		profiles = append(profiles, Quality{
			Name:        res.name,
			Width:       res.w,
			Height:      res.h,
			BitrateKbps: int(rate),
		})
	}
	return profiles
}

// generateHLS создает HLS-плейлисты и сегменты для видео с заданными качествами
// с помощью ffmpeg-go.
// Он принимает URL входного видео, директорию для сохранения выходных файлов и срез качеств.
func (vh *VideoProcess) generateHLS(inputURL string, outputDir string, qualities []Quality) error {
	// TODO: сделать аудио динамическим
	//   Сейчас аудио кодек и аудио_битрейт жестко прописаны.
	n := len(qualities)
	if n == 0 {
		return fmt.Errorf("передан пустой срез qualities")
	}

	var (
		splitLabels []string // будет ["[v0]", "[v1]", "[v2]", ...]
		scaleParts  []string // будет ["[v0]scale=WxH[v0out]", "[v1]scale=WxH[v1out]", ...]
	)
	splitLabels = make([]string, n)
	scaleParts = make([]string, n)
	audioLabels := make([]string, n) // ["[a0]","[a1]","[a2]","[a3]"]

	for i, q := range qualities {
		splitLabels[i] = fmt.Sprintf("[v%d]", i)
		scaleParts[i] = fmt.Sprintf("[v%d]scale=%d:%d[v%dout]", i, q.Width, q.Height, i)
		audioLabels[i] = fmt.Sprintf("[a%d]", i)
	}

	// Конструируем окончательную строку filter_complex
	filterComplex := fmt.Sprintf(
		"[0:v]split=%d%s;%s;[0:a]asplit=%d%s",
		n,
		strings.Join(splitLabels, ""),
		strings.Join(scaleParts, ";"),
		n,
		strings.Join(audioLabels, ""),
	)
	fmt.Printf("filter_complex: %s\n", filterComplex)

	// Map
	mapLabels := make([]string, n)
	for i := 0; i < n; i++ {
		mapLabels[i] = fmt.Sprintf("[v%dout]", i)
		mapLabels = append(mapLabels, fmt.Sprintf("[a%d]", i))
	}
	// mapLabels = append(mapLabels, "0:a")

	fmt.Println("mapLabels:", mapLabels)

	// Формируем сами KwArgs:
	args := ffmpeg_go.KwArgs{
		"filter_complex":       filterComplex,
		"map":                  mapLabels,
		"f":                    "hls",
		"hls_time":             "6",
		"hls_segment_filename": filepath.Join(outputDir, SegmentPattern),
		"hls_playlist_type":    "vod",
	}

	fmt.Println("args:", args)

	for i, q := range qualities {
		// Video кодек для каждого качества.
		// пример ключа: "c:v:0": "libx264"
		keyVideoCodec := fmt.Sprintf("c:v:%d", i)
		args[keyVideoCodec] = AVC

		// битрейт с суффиксом "k" (килобит/с)
		keyBitrate := fmt.Sprintf("b:v:%d", i)
		args[keyBitrate] = fmt.Sprintf("%dk", q.BitrateKbps)

		// Audio кодек для каждого качества.
		keyAudioCodec := fmt.Sprintf("c:a:%d", i)
		args[keyAudioCodec] = AAC

		keyBitrateAudio := fmt.Sprintf("b:a:%d", i)
		// TODO: Сделать битрейт аудио динамическим, если нужно.
		args[keyBitrateAudio] = "128k" // Битрейт аудио фиксированный

	}

	// Для var_stream_map собираем кусок v:i,name:Name_i" и объединяем через пробел.
	var vsEntries []string
	for i, q := range qualities {
		vsEntries = append(vsEntries, fmt.Sprintf("v:%d,a:%d,name:%s", i, i, q.Name))
	}
	args["var_stream_map"] = strings.Join(vsEntries, " ")
	args["master_pl_name"] = MastePLName

	variantPlaylistPattern := filepath.Join(outputDir, VariantPlaylistPattern)

	// Сборка и запуск ffmpeg-команды
	proc := ffmpeg_go.
		Input(inputURL).
		Output(
			variantPlaylistPattern,
			args,
		).ErrorToStdOut()

	if err := proc.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	return nil
}
