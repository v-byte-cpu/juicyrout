package main

type LureService interface {
	ExistsByURL(url string) (bool, error)
}

func NewStaticLureService(lureURLs []string) LureService {
	luresMap := make(map[string]struct{})
	for _, lure := range lureURLs {
		luresMap[lure] = struct{}{}
	}
	return &staticLureService{luresMap}
}

type staticLureService struct {
	lures map[string]struct{}
}

func (s *staticLureService) ExistsByURL(url string) (bool, error) {
	_, exists := s.lures[url]
	return exists, nil
}
