---
title: "AI Layer | AI Layer | AI Layer"
description: "AI layer behavior and integration"
lang: en
version: "0.2"
lastUpdated: 2026-04-10
category: docs
related: [concepts, architecture, modules]
---

# AI Layer | AI Layer | AI Layer

> **AI layer behavior, response formats, and integration guidelines**

---

## Overview

The AI layer is the decision-making component of the Go Assist platform. It's responsible for understanding user inputs, planning execution actions, and reflecting on results to generate user responses. The AI layer follows strict principles: it plans and reflects but never executes actions directly.

---

## AI Layer Architecture

### Two-Phase Processing

The AI layer operates in two distinct phases:

```
Input Processing
       |
Planning Phase (AI generates actions)
       |
Module Execution (Core coordinates)
       |
Reflection Phase (AI generates response)
       |
Output Delivery
```

### Core Components

```go
type AILayer interface {
    // Plan generates actions based on user input
    Plan(ctx context.Context, req PlanRequest) (PlanResponse, error)
    
    // Reflect generates user response based on results
    Reflect(ctx context.Context, req ReflectRequest) (ReflectResponse, error)
    
    // Validate checks if AI response is valid
    Validate(response AIResponse) error
}

type AIEngine struct {
    models        map[string]Model
    router        ModelRouter
    promptManager PromptManager
    cache         Cache
    logger        Logger
    metrics       Metrics
}
```

---

## Planning Phase

### Plan Request Structure

```go
type PlanRequest struct {
    Input      string              `json:"input"`       // User input text
    Context    ExecutionContext    `json:"context"`     // Execution context
    History    []ExecutionResult   `json:"history"`     // Recent execution history
    Constraints []Constraint       `json:"constraints"` // Planning constraints
    UserID     string              `json:"user_id"`     // User identifier
    SessionID  string              `json:"session_id"`  // Session identifier
}
```

### Plan Response Structure

```go
type PlanResponse struct {
    Actions     []Action             `json:"actions"`      // Generated actions
    Confidence  float64              `json:"confidence"`   // Overall confidence
    Reasoning   string               `json:"reasoning"`    // AI reasoning
    Alternatives []AlternativePlan  `json:"alternatives"` // Alternative plans
    Metadata    map[string]interface{} `json:"metadata"`   // Additional metadata
}

type AlternativePlan struct {
    Name        string   `json:"name"`        // Plan name
    Actions     []Action `json:"actions"`      // Alternative actions
    Confidence  float64  `json:"confidence"`   // Alternative confidence
    Reasoning   string   `json:"reasoning"`    // Alternative reasoning
}
```

### Planning Process

#### 1. Input Analysis
```go
func (ai *AIEngine) analyzeInput(ctx context.Context, input string) (InputAnalysis, error) {
    // Extract intent
    intent, err := ai.extractIntent(ctx, input)
    if err != nil {
        return InputAnalysis{}, err
    }
    
    // Extract entities
    entities, err := ai.extractEntities(ctx, input)
    if err != nil {
        return InputAnalysis{}, err
    }
    
    // Determine complexity
    complexity := ai.determineComplexity(input, entities)
    
    return InputAnalysis{
        Intent:     intent,
        Entities:   entities,
        Complexity: complexity,
        Sentiment:  ai.analyzeSentiment(input),
    }, nil
}
```

#### 2. Context Integration
```go
func (ai *AIEngine) integrateContext(ctx context.Context, analysis InputAnalysis, context ExecutionContext) (ContextualAnalysis, error) {
    // Apply user preferences
    preferences := context.Preferences
    
    // Consider timezone
    timezone := context.Timezone
    
    // Check permissions
    permissions := context.Permissions
    
    // Use historical context
    history := context.History
    
    return ContextualAnalysis{
        Input:       analysis,
        Preferences: preferences,
        Timezone:    timezone,
        Permissions: permissions,
        History:     history,
        ContextVars: context.Variables,
    }, nil
}
```

#### 3. Action Generation
```go
func (ai *AIEngine) generateActions(ctx context.Context, analysis ContextualAnalysis) ([]Action, error) {
    // Select appropriate model
    model := ai.router.SelectModel("planning", analysis.Complexity)
    
    // Build planning prompt
    prompt := ai.promptManager.BuildPlanningPrompt(analysis)
    
    // Generate actions
    response, err := model.Generate(ctx, prompt)
    if err != nil {
        return nil, err
    }
    
    // Parse actions from response
    actions, err := ai.parseActions(response)
    if err != nil {
        return nil, err
    }
    
    // Validate actions
    for _, action := range actions {
        if err := ai.validateAction(action, analysis); err != nil {
            return nil, fmt.Errorf("invalid action %s: %w", action.ID, err)
        }
    }
    
    return actions, nil
}
```

### Action Generation Rules

#### 1. Action Structure Rules
```go
type ActionRule struct {
    Module      string   `json:"module"`       // Required module name
    Type        string   `json:"type"`         // Required action type
    Required    []string `json:"required"`     // Required parameters
    Optional    []string `json:"optional"`     // Optional parameters
    Constraints []Constraint `json:"constraints"` // Parameter constraints
}

var ActionRules = map[string]ActionRule{
    "finance.create_transaction": {
        Module:   "finance",
        Type:     "create_transaction",
        Required: []string{"amount", "category"},
        Optional: []string{"currency", "description"},
        Constraints: []Constraint{
            {Field: "amount", Type: "number", Min: 0.01, Max: 10000},
            {Field: "category", Type: "string", Enum: []string{"groceries", "transport", "entertainment"}},
        },
    },
    "calendar.create_event": {
        Module:   "calendar",
        Type:     "create_event",
        Required: []string{"title", "start_time"},
        Optional: []string{"duration", "attendees", "location"},
        Constraints: []Constraint{
            {Field: "start_time", Type: "datetime"},
            {Field: "duration", Type: "duration", Default: "1h"},
        },
    },
}
```

#### 2. Dependency Rules
```go
func (ai *AIEngine) establishDependencies(actions []Action) error {
    // Financial actions should depend on balance checks
    financeActions := filterActions(actions, "finance")
    balanceChecks := filterActions(actions, "finance", "get_balance")
    
    for _, action := range financeActions {
        if action.Type == "create_transaction" && len(balanceChecks) > 0 {
            action.Dependencies = append(action.Dependencies, balanceChecks[0].ID)
        }
    }
    
    // Calendar events should check availability first
    calendarActions := filterActions(actions, "calendar")
    availabilityChecks := filterActions(actions, "calendar", "check_availability")
    
    for _, action := range calendarActions {
        if action.Type == "create_event" && len(availabilityChecks) > 0 {
            action.Dependencies = append(action.Dependencies, availabilityChecks[0].ID)
        }
    }
    
    return nil
}
```

#### 3. Permission Rules
```go
func (ai *AIEngine) checkPermissions(actions []Action, permissions []string) error {
    for _, action := range actions {
        required := getRequiredPermissions(action)
        for _, perm := range required {
            if !contains(permissions, perm) {
                return fmt.Errorf("action %s requires permission %s", action.ID, perm)
            }
        }
    }
    return nil
}

func getRequiredPermissions(action Action) []string {
    switch fmt.Sprintf("%s.%s", action.Module, action.Type) {
    case "finance.create_transaction":
        return []string{"finance:create"}
    case "calendar.create_event":
        return []string{"calendar:create"}
    case "email.send":
        return []string{"email:send"}
    default:
        return []string{}
    }
}
```

---

## Reflection Phase

### Reflect Request Structure

```go
type ReflectRequest struct {
    Input       string           `json:"input"`        // Original user input
    Actions     []Action         `json:"actions"`      // Actions that were executed
    Results     []Result         `json:"results"`      // Execution results
    Context     ExecutionContext `json:"context"`      // Execution context
    ExecutionID string           `json:"execution_id"` // Execution identifier
}
```

### Reflect Response Structure

```go
type ReflectResponse struct {
    Response    string                 `json:"response"`     // User-facing response
    Summary     string                 `json:"summary"`      // Execution summary
    Success     bool                   `json:"success"`      // Overall success
    Insights    []Insight              `json:"insights"`     // AI insights
    Suggestions []Suggestion           `json:"suggestions"`  // Follow-up suggestions
    Metadata    map[string]interface{} `json:"metadata"`     // Additional metadata
}

type Insight struct {
    Type        string `json:"type"`        // Insight type
    Description string `json:"description"` // Insight description
    Confidence  float64 `json:"confidence"` // Insight confidence
}

type Suggestion struct {
    Text        string `json:"text"`        // Suggestion text
    Action      Action `json:"action"`      // Suggested action
    Reasoning   string `json:"reasoning"`   // Suggestion reasoning
}
```

### Reflection Process

#### 1. Result Analysis
```go
func (ai *AIEngine) analyzeResults(ctx context.Context, results []Result) (ResultAnalysis, error) {
    var successful, failed []Result
    
    for _, result := range results {
        if result.Success {
            successful = append(successful, result)
        } else {
            failed = append(failed, result)
        }
    }
    
    // Analyze failure patterns
    failurePatterns := ai.identifyFailurePatterns(failed)
    
    // Extract key metrics
    metrics := ai.extractMetrics(results)
    
    return ResultAnalysis{
        Successful:      successful,
        Failed:         failed,
        SuccessRate:    float64(len(successful)) / float64(len(results)),
        FailurePatterns: failurePatterns,
        Metrics:        metrics,
    }, nil
}
```

#### 2. Response Generation
```go
func (ai *AIEngine) generateResponse(ctx context.Context, req ReflectRequest, analysis ResultAnalysis) (string, error) {
    // Select appropriate model
    model := ai.router.SelectModel("reflection", analysis.Complexity)
    
    // Build reflection prompt
    prompt := ai.promptManager.BuildReflectionPrompt(req, analysis)
    
    // Generate response
    response, err := model.Generate(ctx, prompt)
    if err != nil {
        return "", err
    }
    
    // Validate response
    if err := ai.validateResponse(response, req); err != nil {
        return "", err
    }
    
    return response, nil
}
```

#### 3. Insight Generation
```go
func (ai *AIEngine) generateInsights(ctx context.Context, req ReflectRequest, analysis ResultAnalysis) ([]Insight, error) {
    var insights []Insight
    
    // Financial insights
    if hasFinanceActions(req.Actions) {
        if insight := ai.generateFinancialInsight(analysis); insight != nil {
            insights = append(insights, *insight)
        }
    }
    
    // Calendar insights
    if hasCalendarActions(req.Actions) {
        if insight := ai.generateCalendarInsight(analysis); insight != nil {
            insights = append(insights, *insight)
        }
    }
    
    // Productivity insights
    if insight := ai.generateProductivityInsight(analysis); insight != nil {
        insights = append(insights, *insight)
    }
    
    return insights, nil
}
```

---

## AI Models

### Model Interface

```go
type Model interface {
    Name() string
    Type() ModelType
    Generate(ctx context.Context, prompt Prompt) (Response, error)
    Validate(prompt Prompt) error
    HealthCheck(ctx context.Context) error
}

type ModelType string

const (
    PlanningModel    ModelType = "planning"
    ReflectionModel  ModelType = "reflection"
    AnalysisModel    ModelType = "analysis"
    ValidationModel  ModelType = "validation"
)
```

### Model Router

```go
type ModelRouter struct {
    models map[ModelType][]Model
    strategy RoutingStrategy
}

func (r *ModelRouter) SelectModel(modelType ModelType, complexity Complexity) Model {
    candidates := r.models[modelType]
    
    switch r.strategy {
    case RoundRobin:
        return r.selectRoundRobin(candidates)
    case LoadBalanced:
        return r.selectLoadBalanced(candidates)
    case ComplexityBased:
        return r.selectByComplexity(candidates, complexity)
    default:
        return candidates[0]
    }
}
```

### Model Configuration

```yaml
# config/ai/models.yaml
models:
  planning:
    - name: "gpt-4"
      type: "planning"
      provider: "openai"
      config:
        model: "gpt-4"
        temperature: 0.3
        max_tokens: 2000
    - name: "claude-3"
      type: "planning"
      provider: "anthropic"
      config:
        model: "claude-3-sonnet"
        temperature: 0.2
        max_tokens: 2000
  
  reflection:
    - name: "gpt-3.5-turbo"
      type: "reflection"
      provider: "openai"
      config:
        model: "gpt-3.5-turbo"
        temperature: 0.7
        max_tokens: 1000
```

---

## Prompt Management

### Prompt Templates

```go
type PromptTemplate struct {
    Name     string                 `json:"name"`
    Template string                 `json:"template"`
    Variables map[string]interface{} `json:"variables"`
    Constraints []Constraint        `json:"constraints"`
}

type PromptManager struct {
    templates map[string]PromptTemplate
    renderer  TemplateRenderer
}
```

### Planning Prompt Template

```go
const PlanningPromptTemplate = `
You are an AI planning assistant for the Go Assist platform. Your task is to convert user input into executable actions.

USER INPUT: {{.Input}}
USER CONTEXT: {{.Context}}
AVAILABLE MODULES: {{.Modules}}
RECENT HISTORY: {{.History}}

RULES:
1. Generate ONLY actions that can be executed by available modules
2. Ensure all required parameters are provided
3. Consider user permissions and preferences
4. Establish dependencies between actions when needed
5. Provide confidence score for each action

AVAILABLE ACTIONS:
{{range .ModuleActions}}
- {{.Module}}.{{.Type}}: {{.Description}}
  Required: {{.Required}}
  Optional: {{.Optional}}
{{end}}

RESPONSE FORMAT:
{
  "actions": [
    {
      "module": "module_name",
      "type": "action_type",
      "params": {
        "param1": "value1",
        "param2": "value2"
      },
      "id": "unique_action_id",
      "dependencies": ["action_id_1", "action_id_2"]
    }
  ],
  "confidence": 0.85,
  "reasoning": "Explanation of why these actions were chosen"
}

Generate actions for the user input:
`
```

### Reflection Prompt Template

```go
const ReflectionPromptTemplate = `
You are an AI reflection assistant for the Go Assist platform. Your task to interpret execution results and generate a user-friendly response.

ORIGINAL INPUT: {{.Input}}
EXECUTED ACTIONS: {{.Actions}}
EXECUTION RESULTS: {{.Results}}
USER CONTEXT: {{.Context}}

RESPONSE GUIDELINES:
1. Be clear and concise
2. Explain what was accomplished
3. Highlight any important outcomes
4. Acknowledge failures if any
5. Provide helpful insights when relevant
6. Suggest follow-up actions when appropriate

RESPONSE FORMAT:
{
  "response": "User-friendly response explaining what happened",
  "summary": "Brief summary of execution results",
  "success": true/false,
  "insights": [
    {
      "type": "financial",
      "description": "You spent 15% more on groceries this week",
      "confidence": 0.9
    }
  ],
  "suggestions": [
    {
      "text": "Would you like to set a budget alert?",
      "action": {...},
      "reasoning": "Based on spending patterns"
    }
  ]
}

Generate response for the execution results:
`
```

---

## Response Validation

### Validation Rules

```go
type ValidationRule struct {
    Field      string      `json:"field"`
    Type       string      `json:"type"`
    Required   bool        `json:"required"`
    Constraints []Constraint `json:"constraints"`
}

var PlanValidationRules = []ValidationRule{
    {
        Field:    "actions",
        Type:     "array",
        Required: true,
        Constraints: []Constraint{
            {Field: "min_items", Type: "number", Min: 1},
            {Field: "max_items", Type: "number", Max: 10},
        },
    },
    {
        Field:    "confidence",
        Type:     "number",
        Required: true,
        Constraints: []Constraint{
            {Field: "min", Type: "number", Min: 0.0},
            {Field: "max", Type: "number", Max: 1.0},
        },
    },
}
```

### Validation Implementation

```go
func (ai *AIEngine) validatePlanResponse(response PlanResponse) error {
    // Validate structure
    if len(response.Actions) == 0 {
        return fmt.Errorf("plan must contain at least one action")
    }
    
    // Validate confidence
    if response.Confidence < 0.0 || response.Confidence > 1.0 {
        return fmt.Errorf("confidence must be between 0.0 and 1.0")
    }
    
    // Validate each action
    for _, action := range response.Actions {
        if err := ai.validateAction(action); err != nil {
            return fmt.Errorf("invalid action %s: %w", action.ID, err)
        }
    }
    
    // Validate dependencies
    if err := ai.validateDependencies(response.Actions); err != nil {
        return fmt.Errorf("invalid dependencies: %w", err)
    }
    
    return nil
}

func (ai *AIEngine) validateAction(action Action) error {
    // Check required fields
    if action.Module == "" {
        return fmt.Errorf("module is required")
    }
    
    if action.Type == "" {
        return fmt.Errorf("type is required")
    }
    
    if action.ID == "" {
        return fmt.Errorf("id is required")
    }
    
    // Validate against module registry
    if !ai.registry.IsValidAction(action.Module, action.Type) {
        return fmt.Errorf("unknown action: %s.%s", action.Module, action.Type)
    }
    
    // Validate parameters
    rule := ai.getActionRule(action.Module, action.Type)
    for _, required := range rule.Required {
        if _, ok := action.Params[required]; !ok {
            return fmt.Errorf("required parameter missing: %s", required)
        }
    }
    
    return nil
}
```

---

## Caching Strategy

### Cache Keys

```go
func (ai *AIEngine) getPlanCacheKey(req PlanRequest) string {
    hasher := sha256.New()
    hasher.Write([]byte(req.Input))
    hasher.Write([]byte(req.Context.UserID))
    hasher.Write([]byte(fmt.Sprintf("%v", req.Context.Preferences)))
    
    // Include recent history hash
    historyHash := ai.hashHistory(req.History)
    hasher.Write([]byte(historyHash))
    
    return fmt.Sprintf("plan:%x", hasher.Sum(nil))
}

func (ai *AIEngine) getReflectionCacheKey(req ReflectRequest) string {
    hasher := sha256.New()
    hasher.Write([]byte(req.Input))
    
    // Include results hash
    resultsHash := ai.hashResults(req.Results)
    hasher.Write([]byte(resultsHash))
    
    return fmt.Sprintf("reflection:%x", hasher.Sum(nil))
}
```

### Cache Implementation

```go
func (ai *AIEngine) Plan(ctx context.Context, req PlanRequest) (PlanResponse, error) {
    // Check cache first
    cacheKey := ai.getPlanCacheKey(req)
    if cached, err := ai.cache.Get(ctx, cacheKey); err == nil {
        var response PlanResponse
        if err := json.Unmarshal(cached, &response); err == nil {
            ai.metrics.IncrementCounter("ai.plan.cache_hit")
            return response, nil
        }
    }
    
    // Generate plan
    response, err := ai.generatePlan(ctx, req)
    if err != nil {
        return PlanResponse{}, err
    }
    
    // Cache response
    if data, err := json.Marshal(response); err == nil {
        ai.cache.Set(ctx, cacheKey, data, 5*time.Minute)
    }
    
    ai.metrics.IncrementCounter("ai.plan.cache_miss")
    return response, nil
}
```

---

## Error Handling

### Error Types

```go
type AIError struct {
    Type    ErrorType `json:"type"`
    Message string    `json:"message"`
    Details map[string]interface{} `json:"details"`
}

type ErrorType string

const (
    ModelNotFoundError     ErrorType = "model_not_found"
    PromptValidationError  ErrorType = "prompt_validation_error"
    ResponseValidationError ErrorType = "response_validation_error"
    ModelTimeoutError      ErrorType = "model_timeout"
    InsufficientPermissionsError ErrorType = "insufficient_permissions"
)
```

### Error Recovery

```go
func (ai *AIEngine) PlanWithFallback(ctx context.Context, req PlanRequest) (PlanResponse, error) {
    // Try primary model
    response, err := ai.planWithModel(ctx, req, ai.primaryModel)
    if err == nil {
        return response, nil
    }
    
    ai.logger.Error("Primary model failed", "error", err)
    
    // Try fallback model
    if ai.fallbackModel != nil {
        response, err := ai.planWithModel(ctx, req, ai.fallbackModel)
        if err == nil {
            ai.logger.Info("Fallback model succeeded")
            return response, nil
        }
        
        ai.logger.Error("Fallback model failed", "error", err)
    }
    
    // Return minimal safe response
    return ai.getMinimalPlan(req), nil
}

func (ai *AIEngine) getMinimalPlan(req PlanRequest) PlanResponse {
    // Return a simple clarification request
    return PlanResponse{
        Actions: []Action{
            {
                ID:     "clarify",
                Module: "system",
                Type:   "request_clarification",
                Params: map[string]interface{}{
                    "message": "I need more information to help you with your request.",
                },
            },
        },
        Confidence: 0.5,
        Reasoning:  "Unable to generate specific actions due to AI model limitations",
    }
}
```

---

## Performance Optimization

### Model Selection Strategy

```go
type ModelSelector struct {
    models map[string]Model
    metrics MetricsCollector
}

func (s *ModelSelector) SelectOptimalModel(taskType ModelType, complexity Complexity) Model {
    candidates := s.models[taskType]
    
    var bestModel Model
    var bestScore float64 = -1
    
    for _, model := range candidates {
        score := s.calculateModelScore(model, complexity)
        if score > bestScore {
            bestScore = score
            bestModel = model
        }
    }
    
    return bestModel
}

func (s *ModelSelector) calculateModelScore(model Model, complexity Complexity) float64 {
    // Factor in latency
    latency := s.metrics.GetAverageLatency(model.Name())
    latencyScore := 1.0 / (1.0 + latency.Seconds())
    
    // Factor in success rate
    successRate := s.metrics.GetSuccessRate(model.Name())
    
    // Factor in cost
    cost := s.metrics.GetAverageCost(model.Name())
    costScore := 1.0 / (1.0 + cost)
    
    // Factor in complexity match
    complexityScore := s.calculateComplexityScore(model, complexity)
    
    // Weighted score
    return (latencyScore * 0.3) + (successRate * 0.4) + (costScore * 0.1) + (complexityScore * 0.2)
}
```

### Batch Processing

```go
func (ai *AIEngine) BatchPlan(ctx context.Context, requests []PlanRequest) ([]PlanResponse, error) {
    // Group requests by model
    modelGroups := ai.groupByModel(requests)
    
    var responses []PlanResponse
    
    // Process each group
    for modelName, reqs := range modelGroups {
        model := ai.models[modelName]
        
        // Process batch
        batchResponses, err := ai.processBatch(ctx, model, reqs)
        if err != nil {
            // Fallback to individual processing
            for _, req := range reqs {
                resp, err := ai.Plan(ctx, req)
                if err != nil {
                    resp = ai.getMinimalPlan(req)
                }
                responses = append(responses, resp)
            }
        } else {
            responses = append(responses, batchResponses...)
        }
    }
    
    return responses, nil
}
```

---

## Monitoring and Metrics

### AI-Specific Metrics

```go
type AIMetrics struct {
    // Planning metrics
    PlansGenerated       Counter
    PlanLatency          Histogram
    PlanCacheHitRate     Gauge
    PlanConfidence       Histogram
    
    // Reflection metrics
    ReflectionsGenerated  Counter
    ReflectionLatency    Histogram
    ReflectionCacheHitRate Gauge
    
    // Model metrics
    ModelLatency         HistogramVec   // By model name
    ModelErrors          CounterVec     // By model name
    ModelCost            GaugeVec       // By model name
    
    // Quality metrics
    ActionSuccessRate    Histogram
    ResponseQuality      Histogram
    UserSatisfaction     Gauge
}
```

### Quality Tracking

```go
func (ai *AIEngine) trackQuality(executionID string, req ReflectRequest, resp ReflectResponse) {
    // Track action success rate
    successCount := 0
    for _, result := range req.Results {
        if result.Success {
            successCount++
        }
    }
    successRate := float64(successCount) / float64(len(req.Results))
    
    ai.metrics.ActionSuccessRate.Observe(successRate)
    
    // Track response quality (simplified)
    quality := ai.calculateResponseQuality(resp)
    ai.metrics.ResponseQuality.Observe(quality)
    
    // Log quality metrics
    ai.logger.Info("Quality metrics",
        "execution_id", executionID,
        "success_rate", successRate,
        "response_quality", quality,
        "confidence", resp.Metadata["confidence"],
    )
}
```

---

## Security Considerations

### Input Sanitization

```go
func (ai *AIEngine) sanitizeInput(input string) string {
    // Remove PII
    sanitized := ai.removePII(input)
    
    // Remove malicious content
    sanitized = ai.removeMaliciousContent(sanitized)
    
    // Limit length
    if len(sanitized) > 10000 {
        sanitized = sanitized[:10000]
    }
    
    return sanitized
}

func (ai *AIEngine) removePII(input string) string {
    // Email addresses
    emailRegex := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
    input = emailRegex.ReplaceAllString(input, "[EMAIL]")
    
    // Phone numbers
    phoneRegex := regexp.MustCompile(`\b\d{3}-\d{3}-\d{4}\b`)
    input = phoneRegex.ReplaceAllString(input, "[PHONE]")
    
    // Credit card numbers
    cardRegex := regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)
    input = cardRegex.ReplaceAllString(input, "[CARD]")
    
    return input
}
```

### Access Control

```go
func (ai *AIEngine) checkPermissions(ctx context.Context, req PlanRequest) error {
    permissions := req.Context.Permissions
    
    // Check if user can access requested modules
    for _, action := range req.Actions {
        required := getRequiredPermissions(action)
        for _, perm := range required {
            if !contains(permissions, perm) {
                return fmt.Errorf("insufficient permissions for action %s", action.ID)
            }
        }
    }
    
    // Check rate limits
    if err := ai.checkRateLimits(req.Context.UserID); err != nil {
        return err
    }
    
    return nil
}
```

---

## Testing

### Unit Tests

```go
func TestAIEngine_Plan(t *testing.T) {
    engine := NewTestAIEngine()
    
    tests := []struct {
        name     string
        request  PlanRequest
        expected PlanResponse
        error    bool
    }{
        {
            name: "simple transaction",
            request: PlanRequest{
                Input: "Create a transaction for $50 at grocery store",
                Context: ExecutionContext{
                    UserID: "test_user",
                    Permissions: []string{"finance:create"},
                },
            },
            expected: PlanResponse{
                Actions: []Action{
                    {
                        Module: "finance",
                        Type:   "create_transaction",
                        Params: map[string]interface{}{
                            "amount":   50.0,
                            "category": "groceries",
                        },
                    },
                },
                Confidence: 0.8,
            },
            error: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            response, err := engine.Plan(context.Background(), tt.request)
            
            if tt.error {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, len(tt.expected.Actions), len(response.Actions))
                assert.Greater(t, response.Confidence, 0.0)
            }
        })
    }
}
```

### Integration Tests

```go
func TestAIEngine_Integration(t *testing.T) {
    // Setup real AI engine with test models
    engine := setupTestAIEngine(t)
    defer cleanupTestAIEngine(t, engine)
    
    // Test full planning and reflection cycle
    req := PlanRequest{
        Input: "Send weekly report to finance team and schedule follow-up meeting",
        Context: ExecutionContext{
            UserID:      "test_user",
            Permissions: []string{"email:send", "calendar:create", "finance:read"},
        },
    }
    
    // Plan
    planResp, err := engine.Plan(context.Background(), req)
    assert.NoError(t, err)
    assert.NotEmpty(t, planResp.Actions)
    
    // Mock execution results
    results := mockExecutionResults(planResp.Actions)
    
    // Reflect
    reflectReq := ReflectRequest{
        Input:   req.Input,
        Actions: planResp.Actions,
        Results: results,
        Context: req.Context,
    }
    
    reflectResp, err := engine.Reflect(context.Background(), reflectReq)
    assert.NoError(t, err)
    assert.NotEmpty(t, reflectResp.Response)
    assert.True(t, reflectResp.Success)
}
```

---

## Conclusion

The AI layer is the brain of the Go Assist platform, responsible for intelligent decision-making while maintaining strict boundaries:

- **Never executes actions directly** - only plans and reflects
- **Validates all responses** before returning to core
- **Handles errors gracefully** with fallback strategies
- **Optimizes for performance** through caching and model selection
- **Maintains security** through input sanitization and access control
- **Tracks quality** through comprehensive metrics

Following these guidelines ensures the AI layer remains reliable, performant, and secure while providing intelligent automation capabilities to users.
