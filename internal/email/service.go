package email

import "context"

type emailDataMapper interface {
	FindReady(context.Context) ([]Email, error)
}

type lockDataMapper interface {
	BatchInsert(context.Context, []Lock) ([]Lock, error)
}

type emailClient interface {
	Send(ctx context.Context, raw []byte) error
}

type Service struct {
	emailDataMapper emailDataMapper
	lockDataMapper  lockDataMapper
	emailClient     emailClient
}

func NewService(dataMapper emailDataMapper, lockDataMapper lockDataMapper, emailClient emailClient) *Service {
	return &Service{
		emailDataMapper: dataMapper,
		lockDataMapper:  lockDataMapper,
		emailClient:     emailClient,
	}
}

func (s *Service) LockAndReturnReadyToProcess(ctx context.Context) ([]Email, error) {
	ready, err := s.emailDataMapper.FindReady(ctx)
	if err != nil {
		return nil, err
	}

	var locks []Lock
	for _, email := range ready {
		locks = append(locks, Lock{Id: email.Id})
	}

	locks, err = s.lockDataMapper.BatchInsert(ctx, locks)
	if err != nil {
		return nil, err
	}

	var locked []Email
	for _, email := range ready {
		// I do this because go mod tidy cannot fucking import slices
		ok := false
		for _, lock := range locks {
			if email.Id == lock.Id {
				ok = true
				continue
			}
		}

		if ok {
			locked = append(locked, email)
		}
	}

	return locked, nil
}

func (s *Service) SendAndUnlock(ctx context.Context, email Email) error {
	// TODO read raw file from filesystem
	raw := []byte("TODO implement")

	err := s.emailClient.Send(ctx, raw)
	if err != nil {
		return err
	}

	// TODO implement unlock
	return nil
}
