---
icon: material/chess-knight
---
# Architecture & Design Choices

This project leverages a modern tech stack and adheres to robust design principles to ensure a high-quality, performant, and maintainable application.

## 🐹 Language: Go (Golang)

- Excellent performance, strong concurrency model, fast compilation, static typing, and a strong standard library.

## 🌐 Web Framework: Fiber

- Fast, unopinionated, inspired by Express.js, with great performance and middleware ecosystem.

## 🗄️ ORM: GORM

- Developer-friendly, supports migrations, associations, and a fluent API.

## 🐘 Database: PostgreSQL

- Reliable, feature-rich, strong transactional integrity.

## 🔐 Authentication: JWT & Basic Auth

- JWT for stateless web API auth, Basic Auth for CLI.

## 🚦 Concurrency Safety: sync.Mutex

- Ensures atomicity and data integrity for account balances.

## 📦 Unit of Work Pattern

- Manages groups of business operations as a single transaction.

## 🧹 Code Quality: Qodana

- Static code analysis for code health, integrated into CI/CD.

## ☁️ Deployment: Vercel (Serverless)

- Serverless deployment for automatic scaling and reduced operational overhead.

## ⚡ Event Bus & Webhook-Driven Design

- Internal event bus for decoupled event handling; webhook callbacks for payment completion.
