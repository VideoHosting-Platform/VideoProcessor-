package queue

import "github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"

type Broker interface {
	Consume() (task.VideoTask, error)
	Publish(task.DBUpload) error
}

type TaskConsumer interface {
	Consume() (task.VideoTask, error)
}

type UpdatePublisher interface {
	Publish(task.DBUpload) error
}
