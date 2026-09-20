package app

import "context"

type aiChatGate struct {
	ready chan struct{}
	users int
}

// Serialize one session's transcript + checkpoint updates; other sessions run
// independently. Queued requests remain cancellable and unused gates disappear.
func (a *App) acquireAIChat(ctx context.Context, sessionID string) (func(), error) {
	a.agentRT.runsMu.Lock()
	if a.agentRT.chatGates == nil {
		a.agentRT.chatGates = make(map[string]*aiChatGate)
	}
	gate := a.agentRT.chatGates[sessionID]
	if gate == nil {
		gate = &aiChatGate{ready: make(chan struct{}, 1)}
		a.agentRT.chatGates[sessionID] = gate
	}
	gate.users++
	a.agentRT.runsMu.Unlock()
	cleanup := func() {
		a.agentRT.runsMu.Lock()
		defer a.agentRT.runsMu.Unlock()
		gate.users--
		if gate.users == 0 {
			delete(a.agentRT.chatGates, sessionID)
		}
	}
	select {
	case gate.ready <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-gate.ready
			cleanup()
			return nil, err
		}
		return func() { <-gate.ready; cleanup() }, nil
	case <-ctx.Done():
		cleanup()
		return nil, ctx.Err()
	}
}
