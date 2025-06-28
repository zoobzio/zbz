package core

import (
	"fmt"
	"sync/atomic"
)

// Hook management implementation

func (c *coreImpl[T]) OnBeforeCreate(hook BeforeCreateHook[T]) HookID {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	hookID := HookID(fmt.Sprintf("before_create_%d", atomic.AddInt64(&c.hookCounter, 1)))
	c.beforeCreateHooks[hookID] = hook
	return hookID
}

func (c *coreImpl[T]) OnAfterCreate(hook AfterCreateHook[T]) HookID {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	hookID := HookID(fmt.Sprintf("after_create_%d", atomic.AddInt64(&c.hookCounter, 1)))
	c.afterCreateHooks[hookID] = hook
	return hookID
}

func (c *coreImpl[T]) OnBeforeUpdate(hook BeforeUpdateHook[T]) HookID {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	hookID := HookID(fmt.Sprintf("before_update_%d", atomic.AddInt64(&c.hookCounter, 1)))
	c.beforeUpdateHooks[hookID] = hook
	return hookID
}

func (c *coreImpl[T]) OnAfterUpdate(hook AfterUpdateHook[T]) HookID {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	hookID := HookID(fmt.Sprintf("after_update_%d", atomic.AddInt64(&c.hookCounter, 1)))
	c.afterUpdateHooks[hookID] = hook
	return hookID
}

func (c *coreImpl[T]) OnBeforeDelete(hook BeforeDeleteHook[T]) HookID {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	hookID := HookID(fmt.Sprintf("before_delete_%d", atomic.AddInt64(&c.hookCounter, 1)))
	c.beforeDeleteHooks[hookID] = hook
	return hookID
}

func (c *coreImpl[T]) OnAfterDelete(hook AfterDeleteHook[T]) HookID {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	hookID := HookID(fmt.Sprintf("after_delete_%d", atomic.AddInt64(&c.hookCounter, 1)))
	c.afterDeleteHooks[hookID] = hook
	return hookID
}

func (c *coreImpl[T]) RemoveHook(hookID HookID) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Try to remove from each hook map
	if _, exists := c.beforeCreateHooks[hookID]; exists {
		delete(c.beforeCreateHooks, hookID)
		return nil
	}
	if _, exists := c.afterCreateHooks[hookID]; exists {
		delete(c.afterCreateHooks, hookID)
		return nil
	}
	if _, exists := c.beforeUpdateHooks[hookID]; exists {
		delete(c.beforeUpdateHooks, hookID)
		return nil
	}
	if _, exists := c.afterUpdateHooks[hookID]; exists {
		delete(c.afterUpdateHooks, hookID)
		return nil
	}
	if _, exists := c.beforeDeleteHooks[hookID]; exists {
		delete(c.beforeDeleteHooks, hookID)
		return nil
	}
	if _, exists := c.afterDeleteHooks[hookID]; exists {
		delete(c.afterDeleteHooks, hookID)
		return nil
	}
	
	return fmt.Errorf("hook not found: %s", hookID)
}

// Helper functions for common hook patterns

// OnModelChanged registers hooks for both create and update operations
func (c *coreImpl[T]) OnModelChanged(
	beforeHook func(old, new ZbzModel[T]) error,
	afterHook func(old, new ZbzModel[T]) error,
) (HookID, HookID) {
	var zero ZbzModel[T]
	
	beforeCreateID := c.OnBeforeCreate(func(data ZbzModel[T]) error {
		return beforeHook(zero, data)
	})
	
	c.OnBeforeUpdate(beforeHook)
	
	var afterCreateID HookID
	
	if afterHook != nil {
		afterCreateID = c.OnAfterCreate(func(data ZbzModel[T]) error {
			return afterHook(zero, data)
		})
		
		c.OnAfterUpdate(afterHook)
	}
	
	// Return the first hook ID (caller would need to track others separately)
	return beforeCreateID, afterCreateID
}

// OnAnyChange registers a hook that fires on any data change
func (c *coreImpl[T]) OnAnyChange(hook func(eventType string, old, new ZbzModel[T]) error) []HookID {
	var zero ZbzModel[T]
	
	hookIDs := make([]HookID, 0, 3)
	
	createID := c.OnAfterCreate(func(data ZbzModel[T]) error {
		return hook("created", zero, data)
	})
	hookIDs = append(hookIDs, createID)
	
	updateID := c.OnAfterUpdate(func(old, new ZbzModel[T]) error {
		return hook("updated", old, new)
	})
	hookIDs = append(hookIDs, updateID)
	
	deleteID := c.OnAfterDelete(func(data ZbzModel[T]) error {
		return hook("deleted", data, zero)
	})
	hookIDs = append(hookIDs, deleteID)
	
	return hookIDs
}

// Conditional hooks that only fire when certain conditions are met

// OnFieldChanged registers hooks that fire when a specific field changes
func (c *coreImpl[T]) OnFieldChanged(
	fieldGetter func(ZbzModel[T]) any,
	hook func(old, new ZbzModel[T], oldValue, newValue any) error,
) HookID {
	return c.OnBeforeUpdate(func(old, new ZbzModel[T]) error {
		oldValue := fieldGetter(old)
		newValue := fieldGetter(new)
		
		// Only fire if the field actually changed
		if fmt.Sprintf("%v", oldValue) != fmt.Sprintf("%v", newValue) {
			return hook(old, new, oldValue, newValue)
		}
		
		return nil
	})
}

// Hook chains - allow multiple hooks to be executed in sequence

type HookChain[T any] struct {
	hooks []func(ZbzModel[T]) error
}

func NewHookChain[T any]() *HookChain[T] {
	return &HookChain[T]{
		hooks: make([]func(ZbzModel[T]) error, 0),
	}
}

func (hc *HookChain[T]) Add(hook func(ZbzModel[T]) error) *HookChain[T] {
	hc.hooks = append(hc.hooks, hook)
	return hc
}

func (hc *HookChain[T]) Execute(data ZbzModel[T]) error {
	for _, hook := range hc.hooks {
		if err := hook(data); err != nil {
			return err
		}
	}
	return nil
}

// Register a hook chain as a single hook
func (c *coreImpl[T]) OnBeforeCreateChain(chain *HookChain[T]) HookID {
	return c.OnBeforeCreate(chain.Execute)
}

func (c *coreImpl[T]) OnAfterCreateChain(chain *HookChain[T]) HookID {
	return c.OnAfterCreate(chain.Execute)
}