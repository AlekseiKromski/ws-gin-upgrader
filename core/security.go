package core

type Security struct {
	attemptsCount   int
	attemptsAllowed int
}

func (s *Security) doAttempt() {
	s.attemptsCount++
}

func (s *Security) cleanAttempts() {
	s.attemptsCount = 0
}
