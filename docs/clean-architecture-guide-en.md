# Clean Architecture Learning Guide

This document explores Clean Architecture design principles and implementation methods through your `tomlord.io-backend` project.

## Table of Contents

1. [Clean Architecture Core Concepts](#clean-architecture-core-concepts)
2. [Layer Architecture Explained](#layer-architecture-explained)
3. [Project Architecture Analysis](#project-architecture-analysis)
4. [Implementation Principles and Examples](#implementation-principles-and-examples)
5. [Dependency Injection and Inversion of Control](#dependency-injection-and-inversion-of-control)
6. [Best Practices](#best-practices)
7. [Common Problems and Solutions](#common-problems-and-solutions)

---

## Clean Architecture Core Concepts

### What is Clean Architecture?

Clean Architecture is a software architecture design pattern proposed by Robert C. Martin (Uncle Bob). Its core philosophy is:

1. **Independence**: Frameworks, databases, UI, and external agencies are details that should not affect core business logic
2. **Testability**: Business rules can be tested without UI, database, web server, or other external elements
3. **UI Independence**: UI can be easily changed without changing the rest of the system
4. **Database Independence**: Business rules are not bound to any particular database
5. **External Agency Independence**: Business rules know nothing about the interfaces to the outside world

### Core Principles

#### 1. Dependency Rule
```
Outer layers depend on inner layers, inner layers cannot depend on outer layers
Frameworks → Interface Adapters → Application Business Rules → Enterprise Business Rules
```

#### 2. Separation of Concerns
Each layer has clear responsibilities and should not take on responsibilities of other layers.

#### 3. Inversion of Control
High-level modules should not depend on low-level modules. Both should depend on abstractions.

---

## Layer Architecture Explained

Clean Architecture typically consists of four concentric circle layers:

### 1. Enterprise Business Rules (Entities)
- **Responsibility**: Contains enterprise-wide business rules
- **Characteristics**: Most stable, least changing layer
- **Contents**: Core business logic, domain objects

### 2. Application Business Rules (Use Cases)
- **Responsibility**: Contains application-specific business rules
- **Characteristics**: Orchestrates Entities to complete use cases
- **Contents**: Application services, business processes

### 3. Interface Adapters
- **Responsibility**: Converts data from use case and entity format to external agency format
- **Characteristics**: Contains controllers, gateways, and presenters of MVC architecture
- **Contents**: Controllers, Gateways, Presenters

### 4. Frameworks & Drivers
- **Responsibility**: Combination of frameworks and tools
- **Characteristics**: Outermost layer, contains all details
- **Contents**: Web frameworks, databases, external interfaces

---

## Project Architecture Analysis

Let's analyze how your project embodies Clean Architecture:

### Directory Structure Mapping

```
tomlord.io-backend/
├── cmd/api/                    # Frameworks & Drivers Layer
│   └── main.go                 # Application entry point
├── internal/
│   ├── auth/                   # Application Business Rules Layer
│   │   └── oauth.go           # Authentication use cases
│   ├── services/              # Application Business Rules Layer
│   │   ├── blog.go           # Blog business logic
│   │   └── message.go        # Message business logic
│   ├── middleware/            # Interface Adapters Layer
│   │   └── auth.go           # Authentication middleware
│   ├── server/               # Interface Adapters Layer
│   │   ├── server.go         # Server configuration
│   │   └── routes.go         # Route configuration
│   ├── database/             # Interface Adapters Layer
│   │   └── database.go       # Database adapter
│   ├── config/               # Frameworks & Drivers Layer
│   │   └── config.go         # Configuration management
│   └── db_sqlc/              # Data access layer
└── sqlc/                     # Database related
```

### Layer Analysis

#### 1. Enterprise Business Rules Layer
In your project, this layer is mainly embodied in:
- **Domain Models**: Core entities defined in `internal/db_sqlc/models.go`
- **Business Rules**: Core logic embedded in the Service layer

#### 2. Application Business Rules Layer
```go
// internal/services/blog.go
type BlogService struct {
    dbService database.DBService  // Depends on abstraction, not concrete implementation
}

func (b *BlogService) CreateBlog(ctx context.Context, req CreateBlogRequest) (*BlogInfo, error) {
    // Business logic: validation, transformation, processing
    if req.Lang == "" {
        req.Lang = "zh-tw"  // Business rule: default language
    }
    if req.Duration == "" {
        req.Duration = "5min"  // Business rule: default duration
    }
    
    // Delegate to data layer
    return b.dbService.CreateBlog(ctx, params)
}
```

#### 3. Interface Adapters Layer
```go
// internal/server/routes.go - HTTP Controllers
func (s *Server) createBlogHandler(c *gin.Context) {
    var req services.CreateBlogRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    blog, err := s.blogService.CreateBlog(c.Request.Context(), req)
    // ...
}

// internal/database/database.go - Database Adapter
type DBService interface {
    Health() map[string]string
    Close()
    GetQueries() *db.Queries
    WithTx(ctx context.Context, fn func(q *db.Queries) error) error
}
```

#### 4. Frameworks & Drivers Layer
```go
// cmd/api/main.go - Application entry point
func main() {
    config.Load()                    // Configuration loading
    server, err := server.NewServer() // Dependency injection
    if err != nil {
        panic(fmt.Sprintf("failed to create server: %s", err))
    }
    
    go gracefulShutdown(server, done) // Graceful shutdown
    err = server.ListenAndServe()     // Start service
}
```

---

## Implementation Principles and Examples

### 1. Dependency Injection

#### Problem: Direct dependency on concrete implementation
```go
// ❌ Wrong approach
type BlogService struct {
    db *sql.DB  // Direct dependency on concrete database connection
}
```

#### Solution: Depend on abstract interfaces
```go
// ✅ Correct approach
type BlogService struct {
    dbService database.DBService  // Depend on abstract interface
}

// Interface definition
type DBService interface {
    GetQueries() *db.Queries
    WithTx(ctx context.Context, fn func(q *db.Queries) error) error
}
```

### 2. Inversion of Control

#### Implementation in server.go
```go
func NewServer() (*http.Server, error) {
    // 1. Initialize bottom-layer dependencies
    dbService, err := database.NewDBService(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize database service: %w", err)
    }

    // 2. Inject dependencies into business layer
    authService := auth.NewAuthService(dbService)
    messageService := services.NewMessageService(dbService)
    blogService := services.NewBlogService(dbService)

    // 3. Inject dependencies into middleware layer
    authMiddleware := middleware.NewAuthMiddleware(jwtSecret, authService)

    // 4. Assemble all dependencies
    NewServer := &Server{
        dbService:      dbService,
        authService:    authService,
        messageService: messageService,
        blogService:    blogService,
        authMiddleware: authMiddleware,
    }
    
    return server, nil
}
```

### 3. Data Transformation

#### Using DTOs (Data Transfer Objects)
```go
// Request DTO
type CreateBlogRequest struct {
    Title       string   `json:"title" binding:"required"`
    Slug        string   `json:"slug" binding:"required"`
    Date        string   `json:"date" binding:"required"`
    // ...
}

// Response DTO
type BlogInfo struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Slug        string   `json:"slug"`
    // ...
}

// Conversion function
func (b *BlogService) convertBlogToInfo(blog db.Blog) *BlogInfo {
    return &BlogInfo{
        ID:          uuid.UUID(blog.ID.Bytes).String(),
        Title:       blog.Title,
        Slug:        blog.Slug,
        Date:        blog.Date.Time.Format("2006-01-02"),
        // ...
    }
}
```

### 4. Transaction Management

```go
func (m *MessageService) ToggleMessageThumb(ctx context.Context, messageID, userID string) (bool, error) {
    var toggled bool
    
    // Use transaction to ensure data consistency
    if err := m.dbService.WithTx(ctx, func(qtx *db.Queries) error {
        // Check current state
        thumbed, err := qtx.CheckUserThumbedMessage(ctx, params)
        if err != nil {
            return err
        }
        
        toggled = !thumbed
        
        if thumbed {
            // Remove thumb
            if err := qtx.DeleteMessageThumb(ctx, deleteParams); err != nil {
                return err
            }
        } else {
            // Add thumb
            if _, err := qtx.CreateMessageThumb(ctx, createParams); err != nil {
                return err
            }
        }
        
        // Update count
        return qtx.UpdateMessageThumbCount(ctx, messageUUID)
    }); err != nil {
        return false, err
    }
    
    return toggled, nil
}
```

---

## Dependency Injection and Inversion of Control

### Dependency Injection Pattern

Your project uses the **Constructor Injection** pattern:

```go
// 1. Define interface
type DBService interface {
    Health() map[string]string
    GetQueries() *db.Queries
    WithTx(ctx context.Context, fn func(q *db.Queries) error) error
}

// 2. Service depends on interface
type BlogService struct {
    dbService database.DBService
}

// 3. Inject through constructor
func NewBlogService(dbService database.DBService) *BlogService {
    return &BlogService{
        dbService: dbService,
    }
}
```

### Inversion of Control Container

A simple IoC container is implemented in `server.go`:

```go
func NewServer() (*http.Server, error) {
    // IoC container is responsible for:
    // 1. Creating all dependencies
    // 2. Resolving dependency relationships
    // 3. Injecting dependencies
    // 4. Managing lifecycles
    
    // Create bottom-layer dependencies
    dbService, err := database.NewDBService(ctx)
    
    // Create business layer, inject bottom-layer dependencies
    authService := auth.NewAuthService(dbService)
    blogService := services.NewBlogService(dbService)
    
    // Create middleware layer, inject business layer dependencies
    authMiddleware := middleware.NewAuthMiddleware(jwtSecret, authService)
    
    // Assemble server
    return &Server{...}, nil
}
```

---

## Best Practices

### 1. Interface Design Principles

#### Interface Segregation Principle
```go
// ✅ Good design: Small and focused interfaces
type Querier interface {
    CreateBlog(ctx context.Context, arg CreateBlogParams) (Blog, error)
    GetBlogBySlug(ctx context.Context, slug string) (Blog, error)
    UpdateBlogBySlug(ctx context.Context, arg UpdateBlogBySlugParams) (Blog, error)
}

// ❌ Bad design: Large and comprehensive interface
type Repository interface {
    // Blog related
    CreateBlog(...) error
    GetBlog(...) error
    // User related  
    CreateUser(...) error
    GetUser(...) error
    // Message related
    CreateMessage(...) error
    GetMessage(...) error
    // ... more methods
}
```

#### Depend on Abstractions Principle
```go
// ✅ Depend on abstractions
type MessageService struct {
    dbService database.DBService  // Interface
}

// ❌ Depend on concrete implementation
type MessageService struct {
    db *pgxpool.Pool  // Concrete implementation
}
```

### 2. Error Handling Patterns

```go
func (b *BlogService) CreateBlog(ctx context.Context, req CreateBlogRequest) (*BlogInfo, error) {
    // 1. Input validation
    if req.Title == "" {
        return nil, fmt.Errorf("title is required")
    }
    
    // 2. Business logic processing
    date, err := time.Parse("2006-01-02", req.Date)
    if err != nil {
        return nil, fmt.Errorf("invalid date format: %w", err)  // Wrap error
    }
    
    // 3. Data persistence
    blog, err := queries.CreateBlog(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("failed to create blog: %w", err)  // Wrap error
    }
    
    return b.convertBlogToInfo(blog), nil
}
```

### 3. Test-Friendly Design

#### Using interfaces for easy testing
```go
// Business logic depends on interface
type BlogService struct {
    dbService database.DBService
}

// Use Mock implementation for testing
type MockDBService struct{}

func (m *MockDBService) GetQueries() *db.Queries {
    // Return Mock query object
}

// Test code
func TestBlogService_CreateBlog(t *testing.T) {
    mockDB := &MockDBService{}
    blogService := NewBlogService(mockDB)
    
    // Test business logic without depending on real database
    result, err := blogService.CreateBlog(ctx, request)
    // Assertions...
}
```

### 4. Configuration Management

```go
// internal/config/config.go
func Load() {
    viper.SetDefault("PORT", 8080)
    viper.SetDefault("JWT_SECRET", "your-jwt-secret")
    
    // Environment variables take priority
    viper.AutomaticEnv()
    
    // Configuration file
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    
    if err := viper.ReadInConfig(); err != nil {
        log.Printf("No config file found, using environment variables and defaults")
    }
}
```

---

## Common Problems and Solutions

### 1. Problem: Circular Dependencies

#### Problem Description
```go
// ❌ Circular dependency
package auth
import "project/user"

type AuthService struct {
    userService *user.Service
}

package user
import "project/auth"

type Service struct {
    authService *auth.AuthService
}
```

#### Solution
```go
// ✅ Use interfaces to break circular dependencies
package auth

type UserRepository interface {
    GetUserByID(id string) (*User, error)
}

type AuthService struct {
    userRepo UserRepository  // Depend on abstraction
}

package user

type Service struct {
    // Don't directly depend on AuthService
}

// Inject concrete implementation at assembly layer
```

### 2. Problem: Over-Engineering

#### Problem Description
Creating interfaces and multi-layer abstractions for every small feature.

#### Solution
```go
// ✅ Moderate abstraction: Only create interfaces when needed
type BlogService struct {
    dbService database.DBService  // Need interface: for testing and implementation replacement
    logger    *log.Logger         // Don't need interface: standard library, won't be replaced
}
```

### 3. Problem: Too Much Data Transformation

#### Problem Description
Performing data transformation at every layer, causing code redundancy.

#### Solution
```go
// ✅ Transform at boundaries
func (s *Server) createBlogHandler(c *gin.Context) {
    var req services.CreateBlogRequest
    // HTTP -> Service DTO transformation
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    blog, err := s.blogService.CreateBlog(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // Service DTO -> HTTP response transformation
    c.JSON(http.StatusCreated, blog)
}
```

### 4. Problem: Complex Transaction Management

#### Solution: Use WithTx Pattern
```go
// ✅ Unified transaction management pattern
type DBService interface {
    WithTx(ctx context.Context, fn func(q *db.Queries) error) error
}

func (s *dbService) WithTx(ctx context.Context, fn func(q *db.Queries) error) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return err
    }
    
    qtx := s.queries.WithTx(tx)
    if err := fn(qtx); err != nil {
        _ = tx.Rollback(ctx)
        return err
    }
    return tx.Commit(ctx)
}

// Usage
func (m *MessageService) ComplexOperation(ctx context.Context) error {
    return m.dbService.WithTx(ctx, func(qtx *db.Queries) error {
        // Execute multiple operations within transaction
        if err := qtx.CreateMessage(ctx, params1); err != nil {
            return err
        }
        if err := qtx.UpdateBlog(ctx, params2); err != nil {
            return err
        }
        return nil
    })
}
```

---

## Summary

Your `tomlord.io-backend` project excellently embodies the core principles of Clean Architecture:

### Advantages
1. **Clear layered structure**: Each layer has clear responsibilities
2. **Dependency injection**: Uses interfaces to achieve inversion of control
3. **Testability**: Business logic is decoupled from external dependencies
4. **Maintainability**: Modular design facilitates maintenance and extension

### Improvement Suggestions
1. **Add domain layer**: Consider extracting more business rules into an independent domain layer
2. **Error handling**: Define more specific business error types
3. **Validation logic**: Extract validation logic into a dedicated validation layer
4. **Test coverage**: Add unit tests for core business logic

The goal of Clean Architecture is not to complicate the system, but to make it more flexible, testable, and maintainable. Your project does this well and provides an excellent example for learning Clean Architecture.
