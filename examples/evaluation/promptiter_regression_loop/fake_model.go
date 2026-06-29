//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package main

import (
	"context"
	"fmt"
	"sync"

	"trpc.group/trpc-go/trpc-agent-go/model"
)

// FakeModel is a deterministic model that consumes one queued response group per call.
type FakeModel struct {
	mu            sync.Mutex
	name          string
	callIndex     int
	callCount     int
	responseQueue [][]*model.Response
}

// NewFakeModel creates a deterministic fake model.
func NewFakeModel(name string, responseQueue [][]*model.Response) *FakeModel {
	return &FakeModel{
		name:          name,
		responseQueue: responseQueue,
	}
}

// GenerateContent implements model.Model.
func (m *FakeModel) GenerateContent(
	ctx context.Context,
	request *model.Request,
) (<-chan *model.Response, error) {
	_ = ctx
	_ = request
	m.mu.Lock()
	m.callCount++
	if m.callIndex >= len(m.responseQueue) {
		call := m.callIndex
		queueLen := len(m.responseQueue)
		m.mu.Unlock()
		return nil, fmt.Errorf("fake model %q exhausted: call %d beyond queue %d", m.name, call, queueLen)
	}
	responses := cloneResponses(m.responseQueue[m.callIndex])
	m.callIndex++
	m.mu.Unlock()

	ch := make(chan *model.Response, len(responses))
	go func() {
		defer close(ch)
		for _, response := range responses {
			ch <- response
		}
	}()
	return ch, nil
}

// CallCount returns the total number of GenerateContent calls, including exhausted calls.
func (m *FakeModel) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Info implements model.Model.
func (m *FakeModel) Info() model.Info {
	return model.Info{Name: m.name}
}

func cloneResponses(responses []*model.Response) []*model.Response {
	cloned := make([]*model.Response, 0, len(responses))
	for _, response := range responses {
		if response == nil {
			cloned = append(cloned, nil)
			continue
		}
		cloned = append(cloned, response.Clone())
	}
	return cloned
}
