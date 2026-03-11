package memory

import (
	"sync"
	"testing"

	"github.com/entrhq/forge/pkg/types"
)

func TestConversationMemory_Add(t *testing.T) {
	mem := NewConversationMemory()

	msg := types.NewUserMessage("Hello")
	mem.Add(msg)

	if mem.Count() != 1 {
		t.Errorf("expected count 1, got %d", mem.Count())
	}

	all := mem.GetAll()
	if len(all) != 1 {
		t.Errorf("expected 1 message, got %d", len(all))
	}
	if all[0].Content != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", all[0].Content)
	}
}

func TestConversationMemory_AddNil(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add(nil)

	if mem.Count() != 0 {
		t.Errorf("expected count 0 after adding nil, got %d", mem.Count())
	}
}

func TestConversationMemory_GetAll(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add(types.NewUserMessage("First"))
	mem.Add(types.NewAssistantMessage("Second"))
	mem.Add(types.NewUserMessage("Third"))

	all := mem.GetAll()

	if len(all) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(all))
	}

	if all[0].Content != "First" {
		t.Errorf("expected 'First', got '%s'", all[0].Content)
	}
	if all[1].Content != "Second" {
		t.Errorf("expected 'Second', got '%s'", all[1].Content)
	}
	if all[2].Content != "Third" {
		t.Errorf("expected 'Third', got '%s'", all[2].Content)
	}
}

func TestConversationMemory_GetRecent(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add(types.NewUserMessage("1"))
	mem.Add(types.NewUserMessage("2"))
	mem.Add(types.NewUserMessage("3"))
	mem.Add(types.NewUserMessage("4"))
	mem.Add(types.NewUserMessage("5"))

	t.Run("GetRecent2", func(t *testing.T) {
		recent := mem.GetRecent(2)
		if len(recent) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(recent))
		}
		if recent[0].Content != "4" {
			t.Errorf("expected '4', got '%s'", recent[0].Content)
		}
		if recent[1].Content != "5" {
			t.Errorf("expected '5', got '%s'", recent[1].Content)
		}
	})

	t.Run("GetRecentAll", func(t *testing.T) {
		recent := mem.GetRecent(10)
		if len(recent) != 5 {
			t.Errorf("expected all 5 messages when requesting more than available")
		}
	})

	t.Run("GetRecentZero", func(t *testing.T) {
		recent := mem.GetRecent(0)
		if len(recent) != 0 {
			t.Errorf("expected 0 messages, got %d", len(recent))
		}
	})

	t.Run("GetRecentNegative", func(t *testing.T) {
		recent := mem.GetRecent(-1)
		if len(recent) != 0 {
			t.Errorf("expected 0 messages for negative count, got %d", len(recent))
		}
	})
}

func TestConversationMemory_Clear(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add(types.NewUserMessage("1"))
	mem.Add(types.NewUserMessage("2"))

	if mem.Count() != 2 {
		t.Errorf("expected 2 messages before clear")
	}

	mem.Clear()

	if mem.Count() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", mem.Count())
	}
}

func TestConversationMemory_Count(t *testing.T) {
	mem := NewConversationMemory()

	if mem.Count() != 0 {
		t.Errorf("expected 0 count for new memory")
	}

	mem.Add(types.NewUserMessage("1"))
	if mem.Count() != 1 {
		t.Errorf("expected 1 count after adding 1 message")
	}

	mem.Add(types.NewUserMessage("2"))
	mem.Add(types.NewUserMessage("3"))
	if mem.Count() != 3 {
		t.Errorf("expected 3 count after adding 3 messages")
	}
}

func TestConversationMemory_Prune(t *testing.T) {
	t.Run("PruneWithSystemMessages", func(t *testing.T) {
		mem := NewConversationMemory()

		mem.Add(types.NewSystemMessage("You are helpful"))
		mem.Add(types.NewUserMessage("Hello"))
		mem.Add(types.NewAssistantMessage("Hi there!"))
		mem.Add(types.NewUserMessage("How are you?"))
		mem.Add(types.NewAssistantMessage("I'm doing great!"))

		// Prune to allow only system + 2 recent messages (roughly 50 tokens)
		err := mem.Prune(50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		all := mem.GetAll()

		// Should keep system message + recent messages
		if len(all) < 2 {
			t.Errorf("expected at least system + some messages, got %d", len(all))
		}

		// System message should be preserved
		if all[0].Role != types.RoleSystem {
			t.Error("first message should be system message")
		}
	})

	t.Run("PruneInvalidTokens", func(t *testing.T) {
		mem := NewConversationMemory()
		mem.Add(types.NewUserMessage("Test"))

		err := mem.Prune(0)
		if err == nil {
			t.Error("expected error for zero maxTokens")
		}

		err = mem.Prune(-1)
		if err == nil {
			t.Error("expected error for negative maxTokens")
		}
	})

	t.Run("PruneEmptyMemory", func(t *testing.T) {
		mem := NewConversationMemory()

		err := mem.Prune(100)
		if err != nil {
			t.Errorf("unexpected error pruning empty memory: %v", err)
		}
	})
}

func TestConversationMemory_AddMultiple(t *testing.T) {
	mem := NewConversationMemory()

	messages := []*types.Message{
		types.NewUserMessage("1"),
		types.NewAssistantMessage("2"),
		types.NewUserMessage("3"),
	}

	mem.AddMultiple(messages)

	if mem.Count() != 3 {
		t.Errorf("expected 3 messages, got %d", mem.Count())
	}
}

func TestConversationMemory_GetByRole(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add(types.NewSystemMessage("System"))
	mem.Add(types.NewUserMessage("User1"))
	mem.Add(types.NewAssistantMessage("Assistant1"))
	mem.Add(types.NewUserMessage("User2"))
	mem.Add(types.NewAssistantMessage("Assistant2"))

	t.Run("GetUserMessages", func(t *testing.T) {
		users := mem.GetByRole(types.RoleUser)
		if len(users) != 2 {
			t.Errorf("expected 2 user messages, got %d", len(users))
		}
	})

	t.Run("GetAssistantMessages", func(t *testing.T) {
		assistants := mem.GetByRole(types.RoleAssistant)
		if len(assistants) != 2 {
			t.Errorf("expected 2 assistant messages, got %d", len(assistants))
		}
	})

	t.Run("GetSystemMessages", func(t *testing.T) {
		systems := mem.GetByRole(types.RoleSystem)
		if len(systems) != 1 {
			t.Errorf("expected 1 system message, got %d", len(systems))
		}
	})
}

func TestConversationMemory_ThreadSafety(t *testing.T) {
	mem := NewConversationMemory()

	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			mem.Add(types.NewUserMessage("Message"))
		}(i)
	}

	// Concurrent reads
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mem.GetAll()
			_ = mem.GetRecent(5)
			_ = mem.Count()
		}()
	}

	wg.Wait()

	// Should have 100 messages
	if mem.Count() != 100 {
		t.Errorf("expected 100 messages after concurrent writes, got %d", mem.Count())
	}
}

func TestConversationMemory_GetAllReturnsACopy(t *testing.T) {
	mem := NewConversationMemory()
	mem.Add(types.NewUserMessage("Original"))

	all := mem.GetAll()

	// Append to the returned slice - this should not affect internal storage
	_ = append(all, types.NewUserMessage("New"))

	// Internal storage should still have only 1 message
	if mem.Count() != 1 {
		t.Error("modifying returned slice should not affect internal storage")
	}

	// Get all again - should still be 1 message
	original := mem.GetAll()
	if len(original) != 1 {
		t.Errorf("expected 1 message in internal storage, got %d", len(original))
	}
}
