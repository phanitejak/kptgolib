# sqlitemock

> 💡 A lightweight in-memory SQLite wrapper for unit testing Go applications using GORM.

`sqlitemock` provides a convenient way to spin up a transient SQLite database for testing, seed it with data, simulate errors, and run assertions — all without the need for a real database.

---

## ✨ Features

- ⚡ In-memory SQLite database (`:memory:`)
- ✅ Auto-migration of GORM models
- 🧪 Easy data seeding and clean resets
- ❌ Error simulation for negative test paths
- 🔄 Lightweight CRUD wrappers
- 🔍 Supports foreign key constraints (optional)

---

## 📦 Installation

Add this as a local Go module or clone the repo into your project:

```bash
go get github.com/phanitejak/kptgolib/sqlitemock
