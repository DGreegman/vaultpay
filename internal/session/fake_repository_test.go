package session

import (
	"context"
	"sync"
)

// fakeRepository is an in-memory Repository for tests. It satisfies
// same interface as the Postgres implementation, so the service under test cannot tell the difference

type fakeRepository struct {
	mu sync.Mutex

	// sessions keyed by token hash, which is how the service looks them up
	sessions map[string]*Session

	// failOn lets a test force a specific method to fail, so we can exercise error paths that are hard to trigger by accident
	failOn map[string]error

}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		sessions: make(map[string]*Session),
		failOn: make(map[string]error),
	}
	
}

func (f *fakeRepository) Create(ctx context.Context, s *Session) error{
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.failOn["Create"]; err != nil {
		return err
	}

	// Store a copy. if we stored the pointer, a later change by the caller would silently mutate our "database" - something postgres would never do.
	cp := *s 
	f.sessions[s.TokenHash] = &cp 
	return nil

}

func (f *fakeRepository) GetByTokenHash(ctx context.Context, hash string) (*Session, error)  {

	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.failOn["GetByTokenHash"]; err != nil {
		return nil, err
	}

	s, ok := f.sessions[hash]

	if !ok {
		return nil, ErrNotFound
	}

	cp := *s 
	return &cp, nil
	
}