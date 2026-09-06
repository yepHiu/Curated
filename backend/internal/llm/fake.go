package llm

import "context"

// Streamer is the chat-turn surface used by the agent loop.
type Streamer interface {
	StreamTurn(ctx context.Context, req TurnRequest, onDelta func(string)) (AssistantTurn, error)
}

// ScriptedStreamer replays predetermined turns without a network.
type ScriptedStreamer struct {
	Turns []AssistantTurn
	index int
}

func (s *ScriptedStreamer) StreamTurn(_ context.Context, _ TurnRequest, onDelta func(string)) (AssistantTurn, error) {
	if s.index >= len(s.Turns) {
		turn := AssistantTurn{Content: "I cannot continue."}
		if onDelta != nil && turn.Content != "" {
			onDelta(turn.Content)
		}
		return turn, nil
	}
	turn := s.Turns[s.index]
	s.index++
	if onDelta != nil && turn.Content != "" {
		onDelta(turn.Content)
	}
	return turn, nil
}
