		newTodo := TodoItem{
			Content:   todo.Content,
			Status:    "pending",
			Priority:  todo.Priority,
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}