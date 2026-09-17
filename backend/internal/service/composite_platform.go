package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

type compositeRouteCandidatesContextKey struct{}
type compositeRouteRuntimeContextKey struct{}

type compositeRouteRuntimeState struct {
	mu       sync.RWMutex
	decision CompositeRouteDecision
}

func (s *compositeRouteRuntimeState) set(decision CompositeRouteDecision) {
	if s == nil || !decision.Matched {
		return
	}
	s.mu.Lock()
	s.decision = decision
	s.mu.Unlock()
}

func (s *compositeRouteRuntimeState) get() (CompositeRouteDecision, bool) {
	if s == nil {
		return CompositeRouteDecision{}, false
	}
	s.mu.RLock()
	decision := s.decision
	s.mu.RUnlock()
	return decision, decision.Matched
}

func compositeRouteRuntimeStateFromContext(ctx context.Context) *compositeRouteRuntimeState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(compositeRouteRuntimeContextKey{}).(*compositeRouteRuntimeState)
	return state
}

// WithCompositeRouteCandidates stores the ordered explicit route chain for
// account selection. The first item is the route used by request dispatch.
func WithCompositeRouteCandidates(ctx context.Context, candidates []CompositeRouteDecision) context.Context {
	if ctx == nil || len(candidates) == 0 {
		return ctx
	}
	cloned := append([]CompositeRouteDecision(nil), candidates...)
	ctx = context.WithValue(ctx, compositeRouteCandidatesContextKey{}, cloned)
	if compositeRouteRuntimeStateFromContext(ctx) == nil {
		ctx = context.WithValue(ctx, compositeRouteRuntimeContextKey{}, &compositeRouteRuntimeState{})
	}
	return ctx
}

// WithoutCompositeRouteCandidates scopes a request that needs an explicit
// platform choice (for example GPT6's internal preparation pass) away from
// the public composite execution chain.
func WithoutCompositeRouteCandidates(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, compositeRouteCandidatesContextKey{}, []CompositeRouteDecision(nil))
}

func CompositeRouteCandidatesFromContext(ctx context.Context) []CompositeRouteDecision {
	if ctx == nil {
		return nil
	}
	candidates, _ := ctx.Value(compositeRouteCandidatesContextKey{}).([]CompositeRouteDecision)
	return append([]CompositeRouteDecision(nil), candidates...)
}

// WithResolvedTargetPlatform stores the concrete provider chosen for a request
// made through a composite group.
func WithResolvedTargetPlatform(ctx context.Context, platform string) context.Context {
	platform = strings.TrimSpace(platform)
	if ctx == nil || platform == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxkey.ResolvedTargetPlatform, platform)
}

// ResolvedTargetPlatformFromContext returns the concrete provider chosen for
// the current request, if one was resolved.
func ResolvedTargetPlatformFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if decision, ok := compositeRouteRuntimeStateFromContext(ctx).get(); ok {
		platform := strings.TrimSpace(decision.TargetPlatform)
		return platform, platform != ""
	}
	platform, ok := ctx.Value(ctxkey.ResolvedTargetPlatform).(string)
	platform = strings.TrimSpace(platform)
	if !ok || platform == "" {
		return "", false
	}
	return platform, true
}

func WithCompositeRouteDecision(ctx context.Context, decision CompositeRouteDecision) context.Context {
	if ctx == nil || !decision.Matched {
		return ctx
	}
	if state := compositeRouteRuntimeStateFromContext(ctx); state != nil {
		state.set(decision)
	}
	ctx = WithResolvedTargetPlatform(ctx, decision.TargetPlatform)
	if model := strings.TrimSpace(decision.UpstreamModel); model != "" {
		ctx = context.WithValue(ctx, ctxkey.ResolvedUpstreamModel, model)
	}
	if model := strings.TrimSpace(decision.PublicModel); model != "" {
		ctx = context.WithValue(ctx, ctxkey.RequestedPublicModel, model)
	}
	if source := strings.TrimSpace(decision.Source); source != "" {
		ctx = context.WithValue(ctx, ctxkey.CompositeRouteSource, source)
	}
	return ctx
}

func ResolvedUpstreamModelFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if decision, ok := compositeRouteRuntimeStateFromContext(ctx).get(); ok {
		model := strings.TrimSpace(decision.UpstreamModel)
		return model, model != ""
	}
	model, ok := ctx.Value(ctxkey.ResolvedUpstreamModel).(string)
	model = strings.TrimSpace(model)
	if !ok || model == "" {
		return "", false
	}
	return model, true
}

func RequestedPublicModelFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if decision, ok := compositeRouteRuntimeStateFromContext(ctx).get(); ok {
		model := strings.TrimSpace(decision.PublicModel)
		return model, model != ""
	}
	model, ok := ctx.Value(ctxkey.RequestedPublicModel).(string)
	model = strings.TrimSpace(model)
	if !ok || model == "" {
		return "", false
	}
	return model, true
}

func CompositeRouteSourceFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if decision, ok := compositeRouteRuntimeStateFromContext(ctx).get(); ok {
		source := strings.TrimSpace(decision.Source)
		return source, source != ""
	}
	source, ok := ctx.Value(ctxkey.CompositeRouteSource).(string)
	source = strings.TrimSpace(source)
	if !ok || source == "" {
		return "", false
	}
	return source, true
}

// DetectModelPlatform maps common public model IDs to the concrete provider
// platform used by sub2api. It intentionally returns false for ambiguous model
// names so composite groups fail closed instead of guessing.
func DetectModelPlatform(model string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return "", false
	}

	normalized = strings.TrimPrefix(normalized, "models/")
	if slash := strings.IndexByte(normalized, '/'); slash > 0 {
		provider := strings.TrimSpace(normalized[:slash])
		rest := strings.TrimSpace(normalized[slash+1:])
		switch provider {
		case "anthropic", "claude":
			return PlatformAnthropic, true
		case "openai", "chatgpt":
			return PlatformOpenAI, true
		case "google", "google-ai-studio", "gemini":
			return PlatformGemini, true
		case "xai", "x-ai", "grok":
			return PlatformGrok, true
		case "kimi", "moonshot":
			return PlatformKimi, true
		case "zhipu", "glm", "bigmodel":
			return PlatformZhipu, true
		case "deepseek":
			return PlatformDeepseek, true
		case "minimax":
			return PlatformMiniMax, true
		}
		if rest != "" {
			normalized = strings.TrimPrefix(rest, "models/")
		}
	}

	switch {
	case strings.HasPrefix(normalized, "anthropic.claude-"),
		strings.HasPrefix(normalized, "claude-"):
		return PlatformAnthropic, true
	case strings.HasPrefix(normalized, "gpt-"),
		strings.HasPrefix(normalized, "chatgpt-"),
		strings.HasPrefix(normalized, "codex-"),
		strings.HasPrefix(normalized, "text-embedding-"),
		strings.HasPrefix(normalized, "text-moderation-"),
		strings.HasPrefix(normalized, "omni-moderation-"),
		strings.HasPrefix(normalized, "dall-e-"),
		strings.HasPrefix(normalized, "gpt-image-"),
		strings.HasPrefix(normalized, "tts-"),
		strings.HasPrefix(normalized, "whisper-"),
		hasOpenAISeriesPrefix(normalized):
		return PlatformOpenAI, true
	case strings.HasPrefix(normalized, "gemini-"),
		strings.HasPrefix(normalized, "learnlm-"):
		return PlatformGemini, true
	case normalized == "grok" || strings.HasPrefix(normalized, "grok-"):
		return PlatformGrok, true
	case normalized == "k3",
		normalized == "k3-256k",
		strings.HasPrefix(normalized, "kimi-"),
		strings.HasPrefix(normalized, "moonshot-"):
		return PlatformKimi, true
	case strings.HasPrefix(normalized, "glm-"):
		return PlatformZhipu, true
	case strings.HasPrefix(normalized, "deepseek-"):
		return PlatformDeepseek, true
	case strings.HasPrefix(normalized, "minimax-"),
		strings.HasPrefix(normalized, "abab5"),
		strings.HasPrefix(normalized, "abab6"),
		strings.HasPrefix(normalized, "abab7"):
		return PlatformMiniMax, true
	default:
		return "", false
	}
}

func hasOpenAISeriesPrefix(model string) bool {
	for _, prefix := range []string{"o1", "o3", "o4", "o5"} {
		if model == prefix || strings.HasPrefix(model, prefix+"-") {
			return true
		}
	}
	return false
}

func (s *GatewayService) resolveCompositeRouteDecision(ctx context.Context, group *Group, requestedModel, endpoint string) (CompositeRouteDecision, bool, error) {
	if group == nil || group.Platform != PlatformComposite {
		return CompositeRouteDecision{}, false, nil
	}
	if platform, ok := ResolvedTargetPlatformFromContext(ctx); ok {
		upstreamModel := requestedModel
		if resolvedModel, modelOK := ResolvedUpstreamModelFromContext(ctx); modelOK {
			upstreamModel = resolvedModel
		}
		source := CompositeRouteSourceDetector
		if resolvedSource, sourceOK := CompositeRouteSourceFromContext(ctx); sourceOK {
			source = resolvedSource
		}
		return CompositeRouteDecision{
			Matched:        true,
			Source:         source,
			GroupID:        group.ID,
			PublicModel:    requestedModel,
			TargetPlatform: platform,
			UpstreamModel:  upstreamModel,
			Endpoint:       normalizeCompositeRouteEndpoint(endpoint),
		}, true, nil
	}
	decision, err := s.compositeResolver.Resolve(ctx, group.ID, requestedModel, endpoint)
	if err != nil {
		return decision, false, err
	}
	return decision, decision.Matched, nil
}

func isConcreteRequestPlatform(platform string) bool {
	switch platform {
	case PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok,
		PlatformKimi, PlatformZhipu, PlatformDeepseek, PlatformMiniMax, PlatformOpenCodeGo:
		return true
	default:
		return false
	}
}

func isOpenAICompatibleCompositePlatform(platform string) bool {
	switch platform {
	case PlatformOpenAI, PlatformGrok, PlatformKimi, PlatformZhipu, PlatformDeepseek, PlatformMiniMax, PlatformOpenCodeGo:
		return true
	default:
		return false
	}
}

func compositeRouteCandidatesForSelection(ctx context.Context, openAICompatible bool) []CompositeRouteDecision {
	candidates := CompositeRouteCandidatesFromContext(ctx)
	if len(candidates) < 2 || isOpenAICompatibleCompositePlatform(candidates[0].TargetPlatform) != openAICompatible {
		return nil
	}
	// GPT6 uses DeepSeek only for the private requirements pass. Keep the
	// final execution on the selected GPT6 provider even when a DeepSeek
	// fallback route exists in the composite group.
	gpt6FinalExecution := IsGPT6Model(candidates[0].PublicModel)
	filtered := make([]CompositeRouteDecision, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Matched || isOpenAICompatibleCompositePlatform(candidate.TargetPlatform) != openAICompatible {
			continue
		}
		if gpt6FinalExecution && candidate.TargetPlatform == PlatformDeepseek {
			continue
		}
		if candidates[0].Endpoint == CompositeRouteEndpointGemini &&
			candidate.TargetPlatform != PlatformGemini && candidate.TargetPlatform != PlatformAntigravity {
			continue
		}
		filtered = append(filtered, candidate)
	}
	if len(filtered) < 2 && !(gpt6FinalExecution && len(filtered) > 0) {
		return nil
	}
	return filtered
}

func isCompositeRouteSelectionExhausted(err error) bool {
	return errors.Is(err, ErrNoAvailableAccounts) || errors.Is(err, ErrNoAvailableCompactAccounts)
}

func applyCompositeRouteSelection(account *Account, requestedModel string, decision CompositeRouteDecision) *Account {
	if account == nil {
		return nil
	}
	cloned := *account
	cloned.compositeRouteRequestedModel = strings.TrimSpace(requestedModel)
	cloned.compositeRouteUpstreamModel = strings.TrimSpace(decision.UpstreamModel)
	return &cloned
}

func inheritCompositeRouteSelection(account, selected *Account) *Account {
	if account == nil || selected == nil || selected.compositeRouteUpstreamModel == "" {
		return account
	}
	cloned := *account
	cloned.compositeRouteRequestedModel = selected.compositeRouteRequestedModel
	cloned.compositeRouteUpstreamModel = selected.compositeRouteUpstreamModel
	return &cloned
}
