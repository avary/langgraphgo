package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockComponents(t *testing.T) {
	ctx := context.Background()

	t.Run("MockEmbedder", func(t *testing.T) {
		e := NewMockEmbedder(2)
		assert.Equal(t, 2, e.GetDimension())

		emb, err := e.EmbedDocument(ctx, "test")
		assert.NoError(t, err)
		assert.Len(t, emb, 2)

		embs, err := e.EmbedDocuments(ctx, []string{"test1", "test2"})
		assert.NoError(t, err)
		assert.Len(t, embs, 2)
	})
}
