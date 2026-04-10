# Go Assist - AI-Driven Execution Platform

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://go.dev)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
[![Coverage](https://img.shields.io/badge/coverage-85%25-green.svg)]()
[![Documentation](https://img.shields.io/badge/docs-complete-brightgreen.svg)]()

## Event-driven AI orchestration platform for automated task execution

</div>

---

## Choose Your Language / Select Your Language / Choose Language

<div align="center">

| **English** | **Russian** | **Chinese** |
|-------------|------------|-------------|
| [![English](https://img.shields.io/badge/English-blue.svg)](./README_EN.md) | [![Russian](https://img.shields.io/badge/Russian-red.svg)](./README_RU.md) | [![Chinese](https://img.shields.io/badge/Chinese-orange.svg)](./README_ZH.md) |

</div>

---

## Quick Overview

Go Assist - **production-ready AI execution platform**, which transforms natural language inputs into automated actions through modular, event-driven architecture. The system separates responsibilities between AI planning, core orchestration, and module execution to create scalable and maintainable automation framework.

### Key Features

<div align="center">

```mermaid
graph TD
    A[User Input] --> B[AI Planning]
    B --> C[Module Execution]
    C --> D[AI Reflection]
    D --> E[User Response]
    
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style C fill:#e8f5e8
    style D fill:#fff3e0
    style E fill:#fce4ec
```

</div>

- **AI Orchestration** - Converting natural language understanding into executable actions
- **Modular Architecture** - Isolated execution modules for different domains
- **Event-Driven Design** - Loose coupling between components via EventBus
- **Multi-Step Execution** - Complex workflows with intermediate results
- **Strict Contracts** - Well-defined interfaces between AI, Core, and Modules

### What Makes It Different

Unlike traditional backends, Go Assist follows strict separation of concerns:
- **AI makes decisions**, **Core orchestrates**, **Modules execute**
- No business logic in orchestrator layer
- Event-driven communication instead of direct coupling
- Extensible through plug-in modules

---

## Architecture Overview

```
Input Layer (HTTP, Telegram, Jobs)
    |
CORE Layer (Orchestrator)
    |       \
    |        \
AI Layer     EventBus
    |           |
    |      Modules Layer
    |           |
    |      Data Layer
    |
Output Layer (Response, Events)
```

**Core Components:**
- **Orchestrator**: Event lifecycle management and coordination
- **AI Engine**: Planning and reflection capabilities
- **EventBus**: Decoupled communication between components
- **Modules**: Isolated execution units for specific domains

---

## How It Works

### Execution Pipeline

<div align="center">

```mermaid
sequenceDiagram
    participant U as User
    participant C as Core
    participant AI as AI Layer
    participant M as Modules
    
    U->>C: Natural Language Input
    C->>AI: Plan Actions
    AI-->>C: Structured Actions
    C->>M: Execute Actions
    M-->>C: Execution Results
    C->>AI: Reflect on Results
    AI-->>C: User Response
    C-->>U: Formatted Response
```

</div>

1. **Input Processing** - User input received via HTTP, Telegram, or scheduled jobs
2. **AI Planning** - Core sends input to AI for analysis and action planning
3. **Action Generation** - AI returns structured list of actions to execute
4. **Module Execution** - Core coordinates module execution based on action list
5. **Result Collection** - Modules return execution results to Core
6. **AI Reflection** - Core sends results to AI for final response generation
7. **Response Delivery** - AI-formatted response returned to user

### Example Use Case

**User Request**: "Send weekly report to finance team and schedule follow-up meeting"

**Pipeline Execution**:
```mermaid
sequenceDiagram
    participant U as User
    participant C as Core
    participant AI as AI Layer
    participant M as Modules
    
    U->>C: Send weekly report...
    C->>AI: Plan actions
    AI-->>C: Actions: finance.generate_report, email.send, calendar.schedule
    C->>M: Execute actions
    M-->>C: Results: report generated, email sent, meeting scheduled
    C->>AI: Reflect on results
    AI-->>C: Response: "Weekly report sent..."
    C-->>U: Confirmation with details
```

---

## Quick Start

### Requirements
- Go 1.21+
- PostgreSQL 15+
- Redis 6+
- OpenAI API key (or local AI model)

### Installation

```bash
# Clone repository
git clone https://github.com/ezhigval/Go_Assist.git
cd Go_Assist

# Install dependencies
go mod tidy

# Setup configuration
cp config/config.example.yaml config/config.yaml
cp .env.example .env

# Configure environment variables
# AI_PROVIDER=openai
# AI_API_KEY=your_openai_key
# DB_DSN=postgres://user:pass@localhost:5432/goassist?sslmode=disable
# REDIS_URL=redis://localhost:6379
```

### Running the System

```bash
# Start the core service
go run main.go

# Start with specific transport
go run cmd/ai-assistant/main.go    # HTTP API
go run cmd/telegram-bot/main.go     # Telegram Bot
go run cmd/modulr/main.go          # CLI interface
```

### Verify Installation

```bash
# Test AI orchestration
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{"input": "What modules are available?"}'

# Expected response
{
  "response": "Available modules: finance, calendar, email, tracker, knowledge",
  "actions_executed": 0,
  "execution_time": "1.2s"
}
```

---

## Documentation Structure

<div align="center">

### Complete Documentation Set

| Document | Description | Link |
|----------|-------------|------|
| **Architecture** | System design and data flow | [Architecture_ZH.md](./Architecture_ZH.md) |
| **Concepts** | Core concepts and terminology | [Concepts_ZH.md](./Concepts_ZH.md) |
| **Modules** | Module development guide | [Modules_ZH.md](./Modules_ZH.md) |
| **AI Layer** | AI behavior and integration | [AI_ZH.md](./AI_ZH.md) |

### Documentation Navigation

```mermaid
graph LR
    A[README] --> B[Architecture]
    A --> C[Concepts]
    A --> D[Modules]
    A --> E[AI Layer]
    
    B --> B1[System Design]
    B --> B2[Data Flow]
    B --> B3[Core Components]
    
    C --> C1[Actions]
    C --> C2[Results]
    C --> C3[Context]
    
    D --> D1[Development Guide]
    D --> D2[Best Practices]
    D --> D3[Examples]
    
    E --> E1[Planning Phase]
    E --> E2[Reflection Phase]
    E --> E3[Model Integration]
```

</div>

---

## Key Concepts

### Action
**Action** represents a single, atomic operation that the system should execute:
```go
type Action struct {
    Module string                 `json:"module"`    // Target module name
    Type   string                 `json:"type"`      // Action type within module
    Params map[string]interface{} `json:"params"`    // Action parameters
    ID     string                 `json:"id"`        // Unique action identifier
    Dependencies []string         `json:"dependencies"` // Action dependencies
}
```

### Result
**Result** represents the outcome of executing an Action:
```go
type Result struct {
    ActionID string                 `json:"action_id"` // Corresponding action ID
    Success  bool                   `json:"success"`    // Execution success status
    Data     interface{}            `json:"data"`       // Result data (if successful)
    Error    string                 `json:"error"`      // Error message (if failed)
    Metadata map[string]interface{} `json:"metadata"`   // Additional metadata
    Duration time.Duration         `json:"duration"`   // Execution time
}
```

### Execution Loop
**Execution Loop** transforms user input into automated actions and responses through a 7-phase process.

---

## Contributing

<div align="center">

### We welcome contributions! 

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

#### Development Setup

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Lint code
golangci-lint run

# Format code
go fmt ./...
```

</div>

---

## License

<div align="center">

Licensed under the MIT License. See [LICENSE](./LICENSE) for details.

---

<p align="center">
  <b>AI orchestrates, Core coordinates, Modules execute.</b><br><br>
  <i>Go Assist - Infrastructure for intelligent automation.</i>
</p>

</div>
