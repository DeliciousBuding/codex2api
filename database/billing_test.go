package database

import (
	"math"
	"testing"
)

func TestGPT55StandardPricing(t *testing.T) {
	p := GetModelPricing("gpt-5.5")
	assertFloatEqual(t, p.InputPricePerMToken, 5.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 30.0)
	assertFloatEqual(t, p.CacheReadPricePerMToken, 0.5)
	assertFloatEqual(t, p.InputPricePerMTokenPriority, 12.5)
	assertFloatEqual(t, p.OutputPricePerMTokenPriority, 75.0)
	assertFloatEqual(t, p.CacheReadPricePerMTokenPriority, 1.25)
}

func TestGPT55CalculateCostStandard(t *testing.T) {
	cost := CalculateCost(1000, 500, 200, "gpt-5.5", "")
	assertFloatEqual(t, cost, 0.0191)
}

func TestGPT55CalculateCostPriority(t *testing.T) {
	cost := CalculateCost(1000, 500, 200, "gpt-5.5", "priority")
	assertFloatEqual(t, cost, 0.04775)
}

func TestGPT55CalculateCostFlex(t *testing.T) {
	cost := CalculateCost(1000, 500, 200, "gpt-5.5", "flex")
	assertFloatEqual(t, cost, 0.00955)
}

func TestGPT55ModelNameVariants(t *testing.T) {
	tests := []struct {
		model     string
		wantInput float64
	}{
		{model: "gpt-5.5", wantInput: 5.0},
		{model: "gpt5.5", wantInput: 5.0},
		{model: "gpt5-5", wantInput: 5.0},
		{model: "openai/gpt-5.5", wantInput: 5.0},
		{model: "/models/gpt-5.5", wantInput: 5.0},
		{model: "gpt-5.5-20260401", wantInput: 5.0},
		{model: "gpt-5.5-high", wantInput: 5.0},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := GetModelPricing(tt.model)
			assertFloatEqual(t, p.InputPricePerMToken, tt.wantInput)
		})
	}
}

func TestGPT54Pricing(t *testing.T) {
	p := GetModelPricing("gpt-5.4")
	assertFloatEqual(t, p.InputPricePerMToken, 2.5)
	assertFloatEqual(t, p.OutputPricePerMToken, 15.0)
	assertFloatEqual(t, p.CacheReadPricePerMToken, 0.25)
	assertFloatEqual(t, p.InputPricePerMTokenPriority, 5.0)
	assertFloatEqual(t, p.OutputPricePerMTokenPriority, 30.0)
	assertFloatEqual(t, p.CacheReadPricePerMTokenPriority, 0.5)
}

func TestGPT54MiniPricing(t *testing.T) {
	p := GetModelPricing("gpt-5.4-mini")
	assertFloatEqual(t, p.InputPricePerMToken, 0.75)
	assertFloatEqual(t, p.OutputPricePerMToken, 4.5)
	assertFloatEqual(t, p.CacheReadPricePerMToken, 0.075)
}

func TestGPT54NanoPricing(t *testing.T) {
	p := GetModelPricing("gpt-5.4-nano")
	assertFloatEqual(t, p.InputPricePerMToken, 0.2)
	assertFloatEqual(t, p.OutputPricePerMToken, 1.25)
	assertFloatEqual(t, p.CacheReadPricePerMToken, 0.02)
}

func TestGPT53CodexPricing(t *testing.T) {
	p := GetModelPricing("gpt-5.3-codex")
	assertFloatEqual(t, p.InputPricePerMToken, 1.75)
	assertFloatEqual(t, p.OutputPricePerMToken, 14.0)
	assertFloatEqual(t, p.CacheReadPricePerMToken, 0.175)
	assertFloatEqual(t, p.InputPricePerMTokenPriority, 3.5)
	assertFloatEqual(t, p.OutputPricePerMTokenPriority, 28.0)
	assertFloatEqual(t, p.CacheReadPricePerMTokenPriority, 0.35)
}

func TestGPT53CodexSparkPricing(t *testing.T) {
	p := GetModelPricing("gpt-5.3-codex-spark")
	assertFloatEqual(t, p.InputPricePerMToken, 1.25)
	assertFloatEqual(t, p.OutputPricePerMToken, 10.0)
	assertFloatEqual(t, p.CacheReadPricePerMToken, 0.125)
	assertFloatEqual(t, p.InputPricePerMTokenPriority, 2.5)
	assertFloatEqual(t, p.OutputPricePerMTokenPriority, 20.0)
	assertFloatEqual(t, p.CacheReadPricePerMTokenPriority, 0.25)
}

func TestGPT52Pricing(t *testing.T) {
	p := GetModelPricing("gpt-5.2")
	assertFloatEqual(t, p.InputPricePerMToken, 1.75)
	assertFloatEqual(t, p.OutputPricePerMToken, 14.0)
	assertFloatEqual(t, p.InputPricePerMTokenPriority, 3.5)
	assertFloatEqual(t, p.OutputPricePerMTokenPriority, 28.0)
}

func TestGPT4oMiniPricing(t *testing.T) {
	p := GetModelPricing("gpt-4o-mini")
	assertFloatEqual(t, p.InputPricePerMToken, 0.15)
	assertFloatEqual(t, p.OutputPricePerMToken, 0.6)
}

func TestGPT4oPricing(t *testing.T) {
	p := GetModelPricing("gpt-4o")
	assertFloatEqual(t, p.InputPricePerMToken, 2.5)
	assertFloatEqual(t, p.OutputPricePerMToken, 10.0)
}

func TestGPT4TurboPricing(t *testing.T) {
	p := GetModelPricing("gpt-4-turbo")
	assertFloatEqual(t, p.InputPricePerMToken, 10.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 30.0)
}

func TestGPT4Pricing(t *testing.T) {
	p := GetModelPricing("gpt-4")
	assertFloatEqual(t, p.InputPricePerMToken, 30.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 60.0)
}

func TestGPT35TurboPricing(t *testing.T) {
	p := GetModelPricing("gpt-3.5-turbo")
	assertFloatEqual(t, p.InputPricePerMToken, 0.5)
	assertFloatEqual(t, p.OutputPricePerMToken, 1.5)
}

func TestClaudeOpus4xPricing(t *testing.T) {
	tests := []struct {
		model string
	}{
		{model: "claude-opus-4-7-20260401"},
		{model: "claude-opus-4-6-20250514"},
		{model: "claude-opus-4-5-20250929"},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := GetModelPricing(tt.model)
			assertFloatEqual(t, p.InputPricePerMToken, 5.0)
			assertFloatEqual(t, p.OutputPricePerMToken, 25.0)
		})
	}
}

func TestClaudeOpusOlderPricing(t *testing.T) {
	p := GetModelPricing("claude-opus-4-20250514")
	assertFloatEqual(t, p.InputPricePerMToken, 15.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 75.0)
}

func TestClaudeSonnetPricing(t *testing.T) {
	p := GetModelPricing("claude-sonnet-4-5-20250929")
	assertFloatEqual(t, p.InputPricePerMToken, 3.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 15.0)
}

func TestClaudeHaiku35Pricing(t *testing.T) {
	p := GetModelPricing("claude-3-5-haiku-20241022")
	assertFloatEqual(t, p.InputPricePerMToken, 1.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 5.0)
}

func TestClaudeHaikuOlderPricing(t *testing.T) {
	p := GetModelPricing("claude-3-haiku-20240307")
	assertFloatEqual(t, p.InputPricePerMToken, 0.25)
	assertFloatEqual(t, p.OutputPricePerMToken, 1.25)
}

func TestClaudeGenericFallback(t *testing.T) {
	p := GetModelPricing("claude-unknown-model")
	assertFloatEqual(t, p.InputPricePerMToken, 3.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 15.0)
}

func TestGemini31ProPricing(t *testing.T) {
	tests := []string{"gemini-3.1-pro", "gemini-3-1-pro", "publishers/google/models/gemini-3.1-pro"}
	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			p := GetModelPricing(model)
			assertFloatEqual(t, p.InputPricePerMToken, 2.0)
			assertFloatEqual(t, p.OutputPricePerMToken, 12.0)
		})
	}
}

func TestOpenAIPrefixMatching(t *testing.T) {
	tests := []struct {
		model     string
		wantInput float64
	}{
		{model: "gpt-4o-mini-2024-07-18", wantInput: 0.15},
		{model: "gpt-4o-2024-08-06", wantInput: 2.5},
		{model: "gpt-4-0613", wantInput: 30.0},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := GetModelPricing(tt.model)
			assertFloatEqual(t, p.InputPricePerMToken, tt.wantInput)
		})
	}
}

func TestPrefixMatchingGpt4oMiniBeforeGpt4o(t *testing.T) {
	p := GetModelPricing("gpt-4o-mini")
	assertFloatEqual(t, p.InputPricePerMToken, 0.15)
	// gpt-4o-mini should not match gpt-4o
	p2 := GetModelPricing("gpt-4o")
	assertFloatEqual(t, p2.InputPricePerMToken, 2.5)
}

func TestServiceTierPriorityWithBuiltinPrices(t *testing.T) {
	bd := CalculateCostBreakdown(1000, 500, 200, "gpt-5.5", "priority")
	assertFloatEqual(t, bd.InputPricePerMToken, 12.5)
	assertFloatEqual(t, bd.OutputPricePerMToken, 75.0)
	assertFloatEqual(t, bd.CacheReadPricePerMToken, 1.25)
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 1.0)
}

func TestServiceTierPriorityFallbackMultiplier(t *testing.T) {
	bd := CalculateCostBreakdown(1000, 500, 0, "gpt-4o", "priority")
	assertFloatEqual(t, bd.InputPricePerMToken, 5.0)
	assertFloatEqual(t, bd.OutputPricePerMToken, 20.0)
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 2.0)
}

func TestServiceTierFlexMultiplier(t *testing.T) {
	bd := CalculateCostBreakdown(1000, 500, 0, "gpt-4o", "flex")
	assertFloatEqual(t, bd.InputPricePerMToken, 1.25)
	assertFloatEqual(t, bd.OutputPricePerMToken, 5.0)
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 0.5)
}

func TestServiceTierDefaultMultiplier(t *testing.T) {
	bd := CalculateCostBreakdown(1000, 500, 0, "gpt-4o", "")
	assertFloatEqual(t, bd.InputPricePerMToken, 2.5)
	assertFloatEqual(t, bd.OutputPricePerMToken, 10.0)
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 1.0)
}

func TestServiceTierCaseInsensitive(t *testing.T) {
	bd := CalculateCostBreakdown(1000, 500, 0, "gpt-4o", "PRIORITY")
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 2.0)

	bd = CalculateCostBreakdown(1000, 500, 0, "gpt-4o", "FLEX")
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 0.5)

	bd = CalculateCostBreakdown(1000, 500, 0, "gpt-4o", "  PrioriTY  ")
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 2.0)
}

func TestCachedTokensClamped(t *testing.T) {
	// cached > input: should be clamped to input
	cost := CalculateCost(100, 100, 500, "gpt-5.5", "")
	// effective: uncached=0, cached=100, output=100
	// input: 0 * 5.0/1M = 0, cached: 100 * 0.5/1M = 0.00005, output: 100 * 30.0/1M = 0.003
	// total: 0.00305
	assertFloatEqual(t, cost, 0.00305)
}

func TestCachedTokensNegative(t *testing.T) {
	cost := CalculateCost(1000, 500, -100, "gpt-5.5", "")
	// negative cached treated as 0
	// input: 1000 * 5.0/1M = 0.005, output: 500 * 30.0/1M = 0.015
	// total: 0.02
	assertFloatEqual(t, cost, 0.02)
}

func TestCachedTokensZero(t *testing.T) {
	cost := CalculateCost(1000, 500, 0, "gpt-5.5", "")
	// input: 1000 * 5.0/1M = 0.005, output: 500 * 30.0/1M = 0.015
	// total: 0.02
	assertFloatEqual(t, cost, 0.02)
}

func TestDefaultPricingForUnknownModel(t *testing.T) {
	p := GetModelPricing("nonexistent-model-v42")
	assertFloatEqual(t, p.InputPricePerMToken, 1.0)
	assertFloatEqual(t, p.OutputPricePerMToken, 2.0)
}

func TestCostBreakdownHasAllFields(t *testing.T) {
	bd := CalculateCostBreakdown(1000, 500, 200, "gpt-5.5", "flex")
	// input: (1000-200) * 5.0/1M * 0.5 = 0.002
	// cached: 200 * 0.5/1M * 0.5 = 0.00005
	// output: 500 * 30.0/1M * 0.5 = 0.0075
	// total: 0.00955
	assertFloatEqual(t, bd.InputCost, 0.002)
	assertFloatEqual(t, bd.CacheReadCost, 0.00005)
	assertFloatEqual(t, bd.OutputCost, 0.0075)
	assertFloatEqual(t, bd.TotalCost, 0.00955)
	assertFloatEqual(t, bd.InputPricePerMToken, 2.5)
	assertFloatEqual(t, bd.OutputPricePerMToken, 15.0)
	assertFloatEqual(t, bd.CacheReadPricePerMToken, 0.25)
	assertFloatEqual(t, bd.ServiceTierCostMultiplier, 0.5)
}

func TestCalculateCostEqualsBreakdownTotal(t *testing.T) {
	cost := CalculateCost(1500, 800, 300, "gpt-5.4", "priority")
	bd := CalculateCostBreakdown(1500, 800, 300, "gpt-5.4", "priority")
	assertFloatEqual(t, cost, bd.TotalCost)
}

func TestGPT5ModelFamilyFallback(t *testing.T) {
	// "gpt-5" alone should match gpt-5.4
	p := GetModelPricing("gpt-5")
	assertFloatEqual(t, p.InputPricePerMToken, 2.5)
}

func TestCodexFallback(t *testing.T) {
	p := GetModelPricing("codex-some-variant")
	assertFloatEqual(t, p.InputPricePerMToken, 1.75)
}

func TestGPT53FallbackToCodex(t *testing.T) {
	p := GetModelPricing("gpt-5.3")
	assertFloatEqual(t, p.InputPricePerMToken, 1.75)
}

func TestModelNameWithSpaces(t *testing.T) {
	p := GetModelPricing("  gpt-5.5  ")
	assertFloatEqual(t, p.InputPricePerMToken, 5.0)
}

func TestModelNameWithPublisherPath(t *testing.T) {
	p := GetModelPricing("publishers/anthropic/models/claude-sonnet-4-5-20250929")
	assertFloatEqual(t, p.InputPricePerMToken, 3.0)
}

func assertFloatEqual(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("got %.12f, want %.12f", got, want)
	}
}
