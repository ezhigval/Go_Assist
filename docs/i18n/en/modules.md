---
title: "Modules | Modules | Modules"
description: "Module development guide and best practices"
lang: en
version: "0.2"
lastUpdated: 2026-04-10
category: docs
related: [concepts, architecture, ai-layer]
---

# Modules | Modules | Modules

> **Module development guide and best practices**

---

## Overview

Modules are the execution layer of the Go Assist platform. They implement specific domain capabilities and are completely isolated from each other, communicating only through the EventBus. This guide covers how to develop, test, and deploy modules.

---

## Module Interface

### Core Interface

All modules must implement the `Module` interface:

```go
type Module interface {
    // Execute performs the specified action
    Execute(ctx context.Context, action Action) (Result, error)
    
    // Name returns the module identifier
    Name() string
    
    // Capabilities returns supported action types
    Capabilities() []string
    
    // Validate checks if action is supported and valid
    Validate(action Action) error
    
    // Initialize sets up the module with dependencies
    Initialize(deps Dependencies) error
    
    // Shutdown gracefully stops the module
    Shutdown(ctx context.Context) error
}
```

### Dependencies Interface

Modules receive dependencies through the `Dependencies` interface:

```go
type Dependencies struct {
    EventBus   EventBus
    Logger     Logger
    Config     Config
    Database   Database
    Cache      Cache
    Metrics    Metrics
}
```

---

## Module Structure

### Standard Directory Layout

```
modules/
    finance/
        module.go          # Main module implementation
        actions.go         # Action handlers
        models.go          # Data models
        config.go          # Configuration
        tests/
            module_test.go # Unit tests
            integration_test.go # Integration tests
        README.md          # Module documentation
```

### Basic Module Template

```go
package finance

import (
    "context"
    "fmt"
    "time"
    
    "github.com/go-assist/core"
)

type FinanceModule struct {
    name     string
    deps     core.Dependencies
    config   Config
    db       Database
    cache    Cache
}

func NewFinanceModule() *FinanceModule {
    return &FinanceModule{
        name: "finance",
    }
}

func (f *FinanceModule) Initialize(deps core.Dependencies) error {
    f.deps = deps
    
    // Load configuration
    if err := deps.Config.Get("finance", &f.config); err != nil {
        return fmt.Errorf("failed to load finance config: %w", err)
    }
    
    // Initialize database connection
    f.db = deps.Database
    
    // Initialize cache
    f.cache = deps.Cache
    
    // Subscribe to EventBus events
    deps.EventBus.Subscribe("finance.*", f.handleEvent)
    
    f.deps.Logger.Info("Finance module initialized")
    return nil
}

func (f *FinanceModule) Name() string {
    return f.name
}

func (f *FinanceModule) Capabilities() []string {
    return []string{
        "create_transaction",
        "get_balance",
        "generate_report",
        "list_transactions",
        "update_category",
    }
}

func (f *FinanceModule) Validate(action core.Action) error {
    if action.Module != f.name {
        return fmt.Errorf("module mismatch: expected %s, got %s", f.name, action.Module)
    }
    
    switch action.Type {
    case "create_transaction":
        return f.validateCreateTransaction(action.Params)
    case "get_balance":
        return f.validateGetBalance(action.Params)
    case "generate_report":
        return f.validateGenerateReport(action.Params)
    default:
        return fmt.Errorf("unsupported action type: %s", action.Type)
    }
}

func (f *FinanceModule) Execute(ctx context.Context, action core.Action) (core.Result, error) {
    start := time.Now()
    
    // Validate action
    if err := f.Validate(action); err != nil {
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    err.Error(),
            Duration: time.Since(start),
        }, nil
    }
    
    // Execute action
    var result core.Result
    var err error
    
    switch action.Type {
    case "create_transaction":
        result, err = f.createTransaction(ctx, action)
    case "get_balance":
        result, err = f.getBalance(ctx, action)
    case "generate_report":
        result, err = f.generateReport(ctx, action)
    default:
        err = fmt.Errorf("unsupported action type: %s", action.Type)
    }
    
    // Ensure result has required fields
    if err != nil {
        result = core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    err.Error(),
            Duration: time.Since(start),
        }
    }
    
    result.Duration = time.Since(start)
    
    // Log execution
    f.deps.Logger.Info("Action executed",
        "module", f.name,
        "action", action.Type,
        "success", result.Success,
        "duration", result.Duration,
    )
    
    // Record metrics
    f.deps.Metrics.RecordAction(f.name, action.Type, result.Success, result.Duration)
    
    return result, nil
}

func (f *FinanceModule) Shutdown(ctx context.Context) error {
    f.deps.Logger.Info("Finance module shutting down")
    return nil
}
```

---

## Action Implementation

### Action Handler Pattern

Each action type should have its own handler method:

```go
func (f *FinanceModule) createTransaction(ctx context.Context, action core.Action) (core.Result, error) {
    // Extract parameters
    params := action.Params
    
    amount, ok := params["amount"].(float64)
    if !ok {
        return core.Result{}, fmt.Errorf("invalid amount type")
    }
    
    currency, ok := params["currency"].(string)
    if !ok {
        currency = "USD" // Default currency
    }
    
    category, ok := params["category"].(string)
    if !ok {
        return core.Result{}, fmt.Errorf("category is required")
    }
    
    description, _ := params["description"].(string)
    
    // Create transaction
    txn := &Transaction{
        ID:          generateID(),
        Amount:      amount,
        Currency:    currency,
        Category:    category,
        Description: description,
        CreatedAt:   time.Now(),
        UserID:      getUserID(ctx),
    }
    
    // Save to database
    if err := f.db.SaveTransaction(ctx, txn); err != nil {
        return core.Result{}, fmt.Errorf("failed to save transaction: %w", err)
    }
    
    // Invalidate cache
    f.cache.Delete(fmt.Sprintf("balance:%s", getUserID(ctx)))
    
    // Publish event
    f.deps.EventBus.Publish("finance.transaction.created", map[string]interface{}{
        "transaction_id": txn.ID,
        "amount":        txn.Amount,
        "category":      txn.Category,
    })
    
    return core.Result{
        ActionID: action.ID,
        Success:  true,
        Data: map[string]interface{}{
            "transaction_id": txn.ID,
            "balance_after":  f.calculateBalance(ctx),
            "status":        "completed",
        },
        Metadata: map[string]interface{}{
            "created_at": txn.CreatedAt,
            "account_id":  txn.UserID,
        },
    }, nil
}
```

### Parameter Validation

Each action should validate its parameters:

```go
func (f *FinanceModule) validateCreateTransaction(params map[string]interface{}) error {
    // Check required fields
    if _, ok := params["amount"]; !ok {
        return fmt.Errorf("amount is required")
    }
    
    if _, ok := params["category"]; !ok {
        return fmt.Errorf("category is required")
    }
    
    // Validate amount
    amount, ok := params["amount"].(float64)
    if !ok {
        return fmt.Errorf("amount must be a number")
    }
    
    if amount <= 0 {
        return fmt.Errorf("amount must be positive")
    }
    
    // Validate category
    category, ok := params["category"].(string)
    if !ok {
        return fmt.Errorf("category must be a string")
    }
    
    validCategories := []string{"groceries", "transport", "entertainment", "utilities", "other"}
    if !contains(validCategories, category) {
        return fmt.Errorf("invalid category: %s", category)
    }
    
    // Validate currency if provided
    if currency, ok := params["currency"].(string); ok {
        validCurrencies := []string{"USD", "EUR", "GBP"}
        if !contains(validCurrencies, currency) {
            return fmt.Errorf("invalid currency: %s", currency)
        }
    }
    
    return nil
}
```

---

## Data Models

### Model Structure

Define clear data models for your module:

```go
type Transaction struct {
    ID          string    `json:"id" db:"id"`
    Amount      float64   `json:"amount" db:"amount"`
    Currency    string    `json:"currency" db:"currency"`
    Category    string    `json:"category" db:"category"`
    Description string    `json:"description" db:"description"`
    UserID      string    `json:"user_id" db:"user_id"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Balance struct {
    UserID    string    `json:"user_id"`
    Amount    float64   `json:"amount"`
    Currency  string    `json:"currency"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Report struct {
    Period     string                 `json:"period"`
    UserID     string                 `json:"user_id"`
    Summary    map[string]float64     `json:"summary"`
    Categories map[string]interface{} `json:"categories"`
    GeneratedAt time.Time             `json:"generated_at"`
}
```

### Database Operations

Implement database operations with proper error handling:

```go
func (f *FinanceModule) SaveTransaction(ctx context.Context, txn *Transaction) error {
    query := `
        INSERT INTO transactions (id, amount, currency, category, description, user_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
    
    _, err := f.db.Exec(ctx, query,
        txn.ID, txn.Amount, txn.Currency, txn.Category, txn.Description, txn.UserID, txn.CreatedAt, txn.UpdatedAt,
    )
    
    if err != nil {
        return fmt.Errorf("failed to insert transaction: %w", err)
    }
    
    return nil
}

func (f *FinanceModule) GetTransactions(ctx context.Context, userID string, limit int) ([]Transaction, error) {
    query := `
        SELECT id, amount, currency, category, description, user_id, created_at, updated_at
        FROM transactions
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2
    `
    
    rows, err := f.db.Query(ctx, query, userID, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to query transactions: %w", err)
    }
    defer rows.Close()
    
    var transactions []Transaction
    for rows.Next() {
        var txn Transaction
        if err := rows.Scan(&txn.ID, &txn.Amount, &txn.Currency, &txn.Category, &txn.Description, &txn.UserID, &txn.CreatedAt, &txn.UpdatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan transaction: %w", err)
        }
        transactions = append(transactions, txn)
    }
    
    return transactions, nil
}
```

---

## Configuration

### Configuration Structure

Define configuration for your module:

```go
type Config struct {
    Database DatabaseConfig `yaml:"database"`
    Cache    CacheConfig    `yaml:"cache"`
    Limits   LimitsConfig   `yaml:"limits"`
}

type DatabaseConfig struct {
    TablePrefix string `yaml:"table_prefix"`
    MaxConnections int `yaml:"max_connections"`
}

type CacheConfig struct {
    TTL time.Duration `yaml:"ttl"`
    Prefix string `yaml:"prefix"`
}

type LimitsConfig struct {
    MaxTransactionAmount float64 `yaml:"max_transaction_amount"`
    MaxTransactionsPerDay int    `yaml:"max_transactions_per_day"`
}
```

### Configuration Loading

Load configuration in the Initialize method:

```go
func (f *FinanceModule) Initialize(deps core.Dependencies) error {
    f.deps = deps
    
    // Load module-specific configuration
    var config Config
    if err := deps.Config.GetModuleConfig("finance", &config); err != nil {
        return fmt.Errorf("failed to load finance config: %w", err)
    }
    
    f.config = config
    
    // Apply configuration
    if err := f.applyConfig(); err != nil {
        return fmt.Errorf("failed to apply config: %w", err)
    }
    
    return nil
}

func (f *FinanceModule) applyConfig() error {
    // Set database table prefix
    f.db.SetTablePrefix(f.config.Database.TablePrefix)
    
    // Set cache prefix and TTL
    f.cache.SetPrefix(f.config.Cache.Prefix)
    f.cache.SetTTL(f.config.Cache.TTL)
    
    return nil
}
```

---

## Error Handling

### Error Types

Define specific error types for your module:

```go
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}

type BusinessError struct {
    Operation string
    Reason    string
}

func (e BusinessError) Error() string {
    return fmt.Sprintf("business error in %s: %s", e.Operation, e.Reason)
}

type SystemError struct {
    Component string
    Err       error
}

func (e SystemError) Error() string {
    return fmt.Errorf("system error in %s: %w", e.Component, e.Err)
}
```

### Error Handling Pattern

Handle errors consistently across actions:

```go
func (f *FinanceModule) createTransaction(ctx context.Context, action core.Action) (core.Result, error) {
    // Validate parameters
    if err := f.validateCreateTransaction(action.Params); err != nil {
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    ValidationError{"params", action.Params, err.Error()}.Error(),
        }, nil
    }
    
    // Check business rules
    if err := f.checkBusinessRules(ctx, action.Params); err != nil {
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    BusinessError{"create_transaction", err.Error()}.Error(),
        }, nil
    }
    
    // Execute operation
    if err := f.persistTransaction(ctx, action.Params); err != nil {
        f.deps.Logger.Error("Failed to persist transaction", "error", err)
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    SystemError{"database", err}.Error(),
        }, nil
    }
    
    // Return success
    return core.Result{
        ActionID: action.ID,
        Success:  true,
        Data:     action.Params,
    }, nil
}
```

---

## Testing

### Unit Tests

Write comprehensive unit tests for each action:

```go
package finance

import (
    "context"
    "testing"
    "time"
    
    "github.com/go-assist/core"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestFinanceModule_CreateTransaction(t *testing.T) {
    // Setup
    module := NewFinanceModule()
    
    mockDB := &MockDatabase{}
    mockCache := &MockCache{}
    mockEventBus := &MockEventBus{}
    mockLogger := &MockLogger{}
    mockMetrics := &MockMetrics{}
    
    deps := core.Dependencies{
        Database: mockDB,
        Cache:    mockCache,
        EventBus: mockEventBus,
        Logger:   mockLogger,
        Metrics:  mockMetrics,
    }
    
    err := module.Initialize(deps)
    assert.NoError(t, err)
    
    // Test cases
    tests := []struct {
        name     string
        action   core.Action
        setup    func()
        expected core.Result
    }{
        {
            name: "successful transaction",
            action: core.Action{
                ID:     "test-001",
                Module: "finance",
                Type:   "create_transaction",
                Params: map[string]interface{}{
                    "amount":      100.50,
                    "currency":    "USD",
                    "category":    "groceries",
                    "description": "Weekly shopping",
                },
            },
            setup: func() {
                mockDB.On("SaveTransaction", mock.Anything, mock.AnythingOfType("*finance.Transaction")).Return(nil)
                mockCache.On("Delete", mock.AnythingOfType("string")).Return()
                mockEventBus.On("Publish", "finance.transaction.created", mock.Anything).Return()
            },
            expected: core.Result{
                ActionID: "test-001",
                Success:  true,
                Data: map[string]interface{}{
                    "transaction_id": mock.AnythingOfType("string"),
                    "balance_after":  mock.AnythingOfType("float64"),
                    "status":        "completed",
                },
            },
        },
        {
            name: "invalid amount",
            action: core.Action{
                ID:     "test-002",
                Module: "finance",
                Type:   "create_transaction",
                Params: map[string]interface{}{
                    "amount":   -50.0,
                    "category": "groceries",
                },
            },
            setup:    func() {},
            expected: core.Result{
                ActionID: "test-002",
                Success:  false,
                Error:    "validation error for field amount: amount must be positive",
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            tt.setup()
            
            // Execute
            result, err := module.Execute(context.Background(), tt.action)
            
            // Assert
            assert.NoError(t, err)
            assert.Equal(t, tt.expected.ActionID, result.ActionID)
            assert.Equal(t, tt.expected.Success, result.Success)
            
            if !tt.expected.Success {
                assert.Contains(t, result.Error, tt.expected.Error)
            } else {
                assert.NotNil(t, result.Data)
                assert.Contains(t, result.Data, "transaction_id")
                assert.Contains(t, result.Data, "status")
            }
            
            // Verify mocks
            mockDB.AssertExpectations(t)
            mockCache.AssertExpectations(t)
            mockEventBus.AssertExpectations(t)
        })
    }
}
```

### Integration Tests

Write integration tests that test the full execution flow:

```go
func TestFinanceModule_Integration(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Setup real cache
    cache := setupTestCache(t)
    defer cleanupTestCache(t, cache)
    
    // Setup module
    module := NewFinanceModule()
    deps := core.Dependencies{
        Database: db,
        Cache:    cache,
        EventBus: core.NewEventBus(),
        Logger:   logrus.New(),
        Metrics:  metrics.New(),
    }
    
    err := module.Initialize(deps)
    assert.NoError(t, err)
    
    // Test full workflow
    ctx := context.Background()
    
    // Create transaction
    action := core.Action{
        ID:     "integration-001",
        Module: "finance",
        Type:   "create_transaction",
        Params: map[string]interface{}{
            "amount":      250.75,
            "currency":    "USD",
            "category":    "groceries",
            "description": "Integration test transaction",
        },
    }
    
    result, err := module.Execute(ctx, action)
    assert.NoError(t, err)
    assert.True(t, result.Success)
    
    // Verify transaction was saved
    txnID := result.Data["transaction_id"].(string)
    txn, err := db.GetTransaction(ctx, txnID)
    assert.NoError(t, err)
    assert.Equal(t, 250.75, txn.Amount)
    assert.Equal(t, "groceries", txn.Category)
    
    // Test get balance
    balanceAction := core.Action{
        ID:     "integration-002",
        Module: "finance",
        Type:   "get_balance",
        Params: map[string]interface{}{
            "currency": "USD",
        },
    }
    
    balanceResult, err := module.Execute(ctx, balanceAction)
    assert.NoError(t, err)
    assert.True(t, balanceResult.Success)
    assert.Equal(t, 250.75, balanceResult.Data["balance"])
}
```

---

## Best Practices

### 1. Keep Modules Isolated

- **No direct imports** between modules
- **Communication only** through EventBus
- **Independent configuration** and lifecycle
- **Separate error handling** and logging

### 2. Use Context Properly

```go
func (f *FinanceModule) Execute(ctx context.Context, action core.Action) (core.Result, error) {
    // Extract user ID from context
    userID := getUserID(ctx)
    
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    "operation cancelled",
        }, ctx.Err()
    default:
        // Continue with execution
    }
    
    // Pass context to all operations
    return f.executeWithContext(ctx, action)
}
```

### 3. Implement Proper Logging

```go
func (f *FinanceModule) createTransaction(ctx context.Context, action core.Action) (core.Result, error) {
    logger := f.deps.Logger.WithFields(map[string]interface{}{
        "module":   f.name,
        "action":   action.Type,
        "action_id": action.ID,
        "user_id":  getUserID(ctx),
    })
    
    logger.Info("Creating transaction", "amount", action.Params["amount"])
    
    // ... execution logic ...
    
    if err != nil {
        logger.Error("Failed to create transaction", "error", err)
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    err.Error(),
        }, nil
    }
    
    logger.Info("Transaction created successfully", "transaction_id", txnID)
    return result, nil
}
```

### 4. Use Metrics Effectively

```go
func (f *FinanceModule) Execute(ctx context.Context, action core.Action) (core.Result, error) {
    start := time.Now()
    
    result, err := f.executeAction(ctx, action)
    
    // Record metrics
    f.deps.Metrics.RecordAction(f.name, action.Type, result.Success, time.Since(start))
    
    if result.Success {
        f.deps.Metrics.IncrementCounter("finance.transactions.created")
    } else {
        f.deps.Metrics.IncrementCounter("finance.transactions.failed")
    }
    
    return result, err
}
```

### 5. Handle Timeouts Gracefully

```go
func (f *FinanceModule) Execute(ctx context.Context, action core.Action) (core.Result, error) {
    // Create timeout context
    timeout := 30 * time.Second
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    // Execute with timeout
    result := make(chan core.Result, 1)
    errChan := make(chan error, 1)
    
    go func() {
        r, err := f.executeAction(ctx, action)
        if err != nil {
            errChan <- err
        } else {
            result <- r
        }
    }()
    
    select {
    case r := <-result:
        return r, nil
    case err := <-errChan:
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    err.Error(),
        }, nil
    case <-ctx.Done():
        return core.Result{
            ActionID: action.ID,
            Success:  false,
            Error:    "operation timeout",
        }, ctx.Err()
    }
}
```

### 6. Validate All Inputs

```go
func (f *FinanceModule) Validate(action core.Action) error {
    // Check module name
    if action.Module != f.name {
        return fmt.Errorf("module mismatch: expected %s, got %s", f.name, action.Module)
    }
    
    // Check action type
    if !contains(f.Capabilities(), action.Type) {
        return fmt.Errorf("unsupported action type: %s", action.Type)
    }
    
    // Check action ID
    if action.ID == "" {
        return fmt.Errorf("action ID is required")
    }
    
    // Validate parameters
    switch action.Type {
    case "create_transaction":
        return f.validateCreateTransaction(action.Params)
    case "get_balance":
        return f.validateGetBalance(action.Params)
    // ... other cases
    }
    
    return nil
}
```

### 7. Use Caching Wisely

```go
func (f *FinanceModule) getBalance(ctx context.Context, action core.Action) (core.Result, error) {
    userID := getUserID(ctx)
    currency := action.Params["currency"].(string)
    
    // Check cache first
    cacheKey := fmt.Sprintf("balance:%s:%s", userID, currency)
    if cached, err := f.cache.Get(cacheKey); err == nil {
        return core.Result{
            ActionID: action.ID,
            Success:  true,
            Data:     cached,
            Metadata: map[string]interface{}{
                "source": "cache",
            },
        }, nil
    }
    
    // Calculate from database
    balance, err := f.calculateBalanceFromDB(ctx, userID, currency)
    if err != nil {
        return core.Result{}, err
    }
    
    // Cache result
    f.cache.Set(cacheKey, balance, f.config.Cache.TTL)
    
    return core.Result{
        ActionID: action.ID,
        Success:  true,
        Data: map[string]interface{}{
            "balance": balance,
        },
        Metadata: map[string]interface{}{
            "source": "database",
        },
    }, nil
}
```

### 8. Publish Events for Integration

```go
func (f *FinanceModule) createTransaction(ctx context.Context, action core.Action) (core.Result, error) {
    // ... create transaction logic ...
    
    // Publish events for other modules
    events := []map[string]interface{}{
        {
            "type": "finance.transaction.created",
            "data": map[string]interface{}{
                "transaction_id": txn.ID,
                "amount":        txn.Amount,
                "category":      txn.Category,
                "user_id":       txn.UserID,
            },
        },
        {
            "type": "finance.balance.updated",
            "data": map[string]interface{}{
                "user_id": txn.UserID,
                "balance": newBalance,
            },
        },
    }
    
    for _, event := range events {
        f.deps.EventBus.Publish(event["type"].(string), event["data"])
    }
    
    return result, nil
}
```

---

## Module Registration

### Registry Integration

Register your module with the system:

```go
package main

import (
    "github.com/go-assist/core"
    "github.com/go-assist/modules/finance"
    "github.com/go-assist/modules/calendar"
    "github.com/go-assist/modules/email"
)

func main() {
    // Create orchestrator
    orchestrator := core.NewOrchestrator()
    
    // Register modules
    modules := []core.Module{
        finance.NewFinanceModule(),
        calendar.NewCalendarModule(),
        email.NewEmailModule(),
    }
    
    for _, module := range modules {
        if err := orchestrator.RegisterModule(module); err != nil {
            log.Fatalf("Failed to register module %s: %v", module.Name(), err)
        }
    }
    
    // Start orchestrator
    if err := orchestrator.Start(); err != nil {
        log.Fatalf("Failed to start orchestrator: %v", err)
    }
}
```

### Module Discovery

Implement module discovery for dynamic loading:

```go
func LoadModules(configPath string) ([]core.Module, error) {
    var modules []core.Module
    
    // Read module configuration
    config, err := loadModuleConfig(configPath)
    if err != nil {
        return nil, err
    }
    
    // Load enabled modules
    for _, moduleName := range config.EnabledModules {
        module, err := loadModule(moduleName)
        if err != nil {
            return nil, fmt.Errorf("failed to load module %s: %w", moduleName, err)
        }
        modules = append(modules, module)
    }
    
    return modules, nil
}

func loadModule(name string) (core.Module, error) {
    switch name {
    case "finance":
        return finance.NewFinanceModule(), nil
    case "calendar":
        return calendar.NewCalendarModule(), nil
    case "email":
        return email.NewEmailModule(), nil
    default:
        return nil, fmt.Errorf("unknown module: %s", name)
    }
}
```

---

## Deployment

### Configuration Files

Create configuration files for your module:

```yaml
# config/modules/finance.yaml
database:
  table_prefix: "finance_"
  max_connections: 10

cache:
  ttl: "5m"
  prefix: "finance:"

limits:
  max_transaction_amount: 10000.00
  max_transactions_per_day: 100

logging:
  level: "info"
  format: "json"
```

### Health Checks

Implement health checks for your module:

```go
func (f *FinanceModule) HealthCheck(ctx context.Context) error {
    // Check database connection
    if err := f.db.Ping(ctx); err != nil {
        return fmt.Errorf("database connection failed: %w", err)
    }
    
    // Check cache connection
    if err := f.cache.Ping(ctx); err != nil {
        return fmt.Errorf("cache connection failed: %w", err)
    }
    
    return nil
}
```

### Graceful Shutdown

Implement proper shutdown:

```go
func (f *FinanceModule) Shutdown(ctx context.Context) error {
    f.deps.Logger.Info("Finance module shutting down")
    
    // Stop accepting new actions
    f.stopAcceptingActions()
    
    // Wait for ongoing actions to complete
    f.waitForOngoingActions(ctx)
    
    // Close database connections
    if err := f.db.Close(); err != nil {
        f.deps.Logger.Error("Failed to close database connection", "error", err)
    }
    
    // Close cache connections
    if err := f.cache.Close(); err != nil {
        f.deps.Logger.Error("Failed to close cache connection", "error", err)
    }
    
    return nil
}
```

---

## Performance Optimization

### Connection Pooling

Use connection pooling for database operations:

```go
func (f *FinanceModule) Initialize(deps core.Dependencies) error {
    // Setup database connection pool
    poolConfig := &sql.DBConfig{
        MaxOpenConns: f.config.Database.MaxConnections,
        MaxIdleConns: f.config.Database.MaxConnections / 2,
        ConnMaxLifetime: time.Hour,
    }
    
    f.db = NewPooledDatabase(poolConfig)
    
    return nil
}
```

### Batch Operations

Implement batch operations for better performance:

```go
func (f *FinanceModule) createTransactions(ctx context.Context, actions []core.Action) []core.Result {
    var results []core.Result
    
    // Group transactions by user
    transactionsByUser := make(map[string][]Transaction)
    for _, action := range actions {
        // Extract transaction data
        txn := extractTransaction(action)
        userID := getUserID(ctx)
        transactionsByUser[userID] = append(transactionsByUser[userID], txn)
    }
    
    // Batch insert by user
    for userID, transactions := range transactionsByUser {
        if err := f.db.BatchInsertTransactions(ctx, userID, transactions); err != nil {
            // Create error results for this batch
            for _, action := range actions {
                if getUserID(ctx) == userID {
                    results = append(results, core.Result{
                        ActionID: action.ID,
                        Success:  false,
                        Error:    err.Error(),
                    })
                }
            }
        } else {
            // Create success results for this batch
            for _, action := range actions {
                if getUserID(ctx) == userID {
                    results = append(results, core.Result{
                        ActionID: action.ID,
                        Success:  true,
                        Data: map[string]interface{}{
                            "transaction_id": generateID(),
                            "status":        "completed",
                        },
                    })
                }
            }
        }
    }
    
    return results
}
```

---

## Security Considerations

### Input Sanitization

Sanitize all user inputs:

```go
func (f *FinanceModule) sanitizeCreateTransaction(params map[string]interface{}) error {
    // Sanitize amount
    if amount, ok := params["amount"].(float64); ok {
        if amount > f.config.Limits.MaxTransactionAmount {
            return fmt.Errorf("amount exceeds maximum limit")
        }
    }
    
    // Sanitize description
    if description, ok := params["description"].(string); ok {
        params["description"] = sanitizeString(description)
    }
    
    // Validate category against allowed list
    if category, ok := params["category"].(string); ok {
        if !contains(f.getAllowedCategories(), category) {
            return fmt.Errorf("invalid category")
        }
    }
    
    return nil
}
```

### Access Control

Implement access control checks:

```go
func (f *FinanceModule) checkPermissions(ctx context.Context, action core.Action) error {
    userID := getUserID(ctx)
    permissions := getUserPermissions(ctx)
    
    switch action.Type {
    case "create_transaction":
        if !contains(permissions, "finance:create") {
            return fmt.Errorf("insufficient permissions to create transaction")
        }
    case "get_balance":
        if !contains(permissions, "finance:read") {
            return fmt.Errorf("insufficient permissions to read balance")
        }
    // ... other cases
    }
    
    return nil
}
```

### Audit Logging

Implement audit logging for sensitive operations:

```go
func (f *FinanceModule) createTransaction(ctx context.Context, action core.Action) (core.Result, error) {
    // Log audit event
    f.deps.AuditLogger.Info("Transaction creation attempt",
        "user_id", getUserID(ctx),
        "amount", action.Params["amount"],
        "category", action.Params["category"],
        "ip_address", getClientIP(ctx),
    )
    
    // ... execute transaction ...
    
    if result.Success {
        f.deps.AuditLogger.Info("Transaction created successfully",
            "user_id", getUserID(ctx),
            "transaction_id", result.Data["transaction_id"],
            "amount", action.Params["amount"],
        )
    } else {
        f.deps.AuditLogger.Warn("Transaction creation failed",
            "user_id", getUserID(ctx),
            "error", result.Error,
        )
    }
    
    return result, nil
}
```

---

## Monitoring and Observability

### Custom Metrics

Define custom metrics for your module:

```go
func (f *FinanceModule) setupMetrics() {
    // Transaction metrics
    f.deps.Metrics.RegisterCounter("finance.transactions.created", "Number of transactions created")
    f.deps.Metrics.RegisterCounter("finance.transactions.failed", "Number of failed transactions")
    f.deps.Metrics.RegisterHistogram("finance.transaction.amount", "Transaction amounts")
    f.deps.Metrics.RegisterHistogram("finance.transaction.duration", "Transaction processing duration")
    
    // Balance metrics
    f.deps.Metrics.RegisterGauge("finance.balance.current", "Current balance by user")
    f.deps.Metrics.RegisterHistogram("finance.balance.query_duration", "Balance query duration")
}
```

### Distributed Tracing

Add distributed tracing:

```go
func (f *FinanceModule) createTransaction(ctx context.Context, action core.Action) (core.Result, error) {
    // Start span
    span, ctx := f.deps.Tracer.StartSpan(ctx, "finance.create_transaction")
    defer span.Finish()
    
    // Add span attributes
    span.SetAttributes(
        attribute.String("module", "finance"),
        attribute.String("action", "create_transaction"),
        attribute.String("user_id", getUserID(ctx)),
    )
    
    // ... execute transaction ...
    
    if result.Success {
        span.SetAttributes(
            attribute.Bool("success", true),
            attribute.String("transaction_id", result.Data["transaction_id"].(string)),
        )
    } else {
        span.SetAttributes(
            attribute.Bool("success", false),
            attribute.String("error", result.Error),
        )
        span.SetStatus(codes.Internal, result.Error)
    }
    
    return result, nil
}
```

---

## Conclusion

Following these best practices will ensure your modules are:

- **Reliable**: Proper error handling and testing
- **Performant**: Efficient use of resources and caching
- **Secure**: Input validation and access control
- **Observable**: Comprehensive logging and metrics
- **Maintainable**: Clean code structure and documentation

Remember that modules are the foundation of the Go Assist platform, and their quality directly impacts the overall system performance and user experience.
