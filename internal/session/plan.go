package session


type planState struct {
	Request string
	Steps   []toolCall
	Index   int
}

func (s *Session) getPlan(request string) (planState, bool) {
	if s == nil {
		return planState{}, false
	}
	s.planMu.Lock()
	defer s.planMu.Unlock()
	if s.plan == nil || s.plan.Request != request {
		return planState{}, false
	}
	return *s.plan, true
}

func (s *Session) setPlan(state planState) {
	if s == nil {
		return
	}
	s.planMu.Lock()
	defer s.planMu.Unlock()
	s.plan = &state
}

func (s *Session) advancePlan() {
	if s == nil {
		return
	}
	s.planMu.Lock()
	defer s.planMu.Unlock()
	if s.plan == nil {
		return
	}
	if s.plan.Index < len(s.plan.Steps) {
		s.plan.Index++
	}
	if s.plan.Index >= len(s.plan.Steps) {
		s.plan = nil
	}
}

func (s *Session) clearPlan() {
	if s == nil {
		return
	}
	s.planMu.Lock()
	defer s.planMu.Unlock()
	s.plan = nil
}
