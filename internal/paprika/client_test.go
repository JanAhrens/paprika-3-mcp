package paprika_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soggycactus/paprika-3-mcp/internal/paprika"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	username := os.Getenv("PAPRIKA_USERNAME")
	password := os.Getenv("PAPRIKA_PASSWORD")
	client, err := paprika.NewClient(username, password, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	uuid := strings.ToUpper(uuid.NewString())

	testRecipe := paprika.Recipe{
		UID:         uuid,
		Name:        fmt.Sprintf("Test Recipe - %s", time.Now().Format(time.RFC3339)),
		Notes:       "Notes",
		Directions:  "Directions",
		Ingredients: "Ingredients",
		Servings:    "Servings",
		Source:      "Source",
		SourceURL:   "URL",
		Categories:  []string{},
	}
	_, err = client.CreateRecipe(ctx, testRecipe)
	require.NoError(t, err)

	recipe, err := client.GetRecipe(ctx, testRecipe.UID)
	require.NoError(t, err)
	assert.Equal(t, testRecipe.UID, recipe.UID)
	assert.Equal(t, testRecipe.Name, recipe.Name)
	assert.Equal(t, testRecipe.Notes, recipe.Notes)
	assert.Equal(t, testRecipe.Directions, recipe.Directions)
	assert.Equal(t, testRecipe.Ingredients, recipe.Ingredients)
	assert.Equal(t, testRecipe.Servings, recipe.Servings)
	assert.Equal(t, testRecipe.Source, recipe.Source)
	assert.Equal(t, testRecipe.SourceURL, recipe.SourceURL)
	assert.Equal(t, testRecipe.Categories, recipe.Categories)

	t.Logf("Created and fetched recipe: %+v", recipe)

	newDescription := "Updated Description"
	recipe.Description = newDescription
	recipe, err = client.UpdateRecipe(ctx, *recipe)
	require.NoError(t, err)
	assert.Equal(t, newDescription, recipe.Description)
	assert.Equal(t, testRecipe.UID, recipe.UID)
	assert.Equal(t, testRecipe.Name, recipe.Name)
	assert.Equal(t, testRecipe.Notes, recipe.Notes)
	assert.Equal(t, testRecipe.Directions, recipe.Directions)
	assert.Equal(t, testRecipe.Ingredients, recipe.Ingredients)
	assert.Equal(t, testRecipe.Servings, recipe.Servings)
	assert.Equal(t, testRecipe.Source, recipe.Source)
	assert.Equal(t, testRecipe.SourceURL, recipe.SourceURL)
	assert.Equal(t, testRecipe.Categories, recipe.Categories)

	t.Logf("Updated recipe: %+v", recipe)

	_, err = client.DeleteRecipe(ctx, *recipe)
	require.NoError(t, err)
	t.Logf("Deleted recipe: %s", recipe.Name)

	recipes, err := client.ListRecipes(ctx)
	require.NoError(t, err)

	for _, recipe := range recipes.Result {
		r, err := client.GetRecipe(ctx, recipe.UID)
		require.NoError(t, err)

		t.Logf("Recipe: %s", r.Name)
	}
}
