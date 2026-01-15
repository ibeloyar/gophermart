package service

type Storage interface {
	Ping() error
}

type Service struct {
	storage Storage
}

func New(s Storage) *Service {
	return &Service{
		storage: s,
	}
}

func (s *Service) Ping() error {
	err := s.storage.Ping()
	if err != nil {
		return err
	}

	return nil
}
