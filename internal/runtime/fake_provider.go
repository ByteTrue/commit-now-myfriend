package runtime

type FakeProviderStep struct {
	ToolCalls []ToolCallRequest
	Err       error
}

type FakeProviderTracer struct {
	Steps           []FakeProviderStep
	ReceivedResults []ToolCallResult
	index           int
}

func (p *FakeProviderTracer) NextToolCalls(results []ToolCallResult) (ProviderTurn, error) {
	p.ReceivedResults = append(p.ReceivedResults, results...)
	if p.index >= len(p.Steps) {
		return ProviderTurn{}, nil
	}
	step := p.Steps[p.index]
	p.index++
	if step.Err != nil {
		return ProviderTurn{}, step.Err
	}
	return ProviderTurn{ToolCalls: step.ToolCalls}, nil
}
