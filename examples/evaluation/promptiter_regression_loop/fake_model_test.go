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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-agent-go/model"
)

func TestFakeModelConsumesOneQueuedCallPerGenerateContent(t *testing.T) {
	fake := NewFakeModel("candidate", [][]*model.Response{
		{textResponse("first")},
		{textResponse("second")},
	})

	first, err := drainResponses(context.Background(), fake)
	require.NoError(t, err)
	second, err := drainResponses(context.Background(), fake)
	require.NoError(t, err)
	_, err = drainResponses(context.Background(), fake)

	require.Error(t, err)
	assert.Equal(t, "first", first[0].Choices[0].Message.Content)
	assert.Equal(t, "second", second[0].Choices[0].Message.Content)
	assert.Equal(t, 3, fake.CallCount())
	assert.Equal(t, "candidate", fake.Info().Name)
}

func drainResponses(ctx context.Context, fake *FakeModel) ([]*model.Response, error) {
	ch, err := fake.GenerateContent(ctx, &model.Request{})
	if err != nil {
		return nil, err
	}
	var responses []*model.Response
	for response := range ch {
		responses = append(responses, response)
	}
	return responses, nil
}

func textResponse(content string) *model.Response {
	finish := "stop"
	return &model.Response{
		Object: model.ObjectTypeChatCompletion,
		Model:  "fake",
		Choices: []model.Choice{{
			Index: 0,
			Message: model.Message{
				Role:    model.RoleAssistant,
				Content: content,
			},
			FinishReason: &finish,
		}},
		Done: true,
	}
}
