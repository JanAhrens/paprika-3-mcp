package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/soggycactus/paprika-3-mcp/internal/paprika"
)

func extractUID(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	// Expect path to be in the format "/recipes/{uid}"
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid URI: %s", uri)
	}

	return parts[0], nil
}

func main() {
	username := flag.String("username", "", "Paprika 3 username (email)")
	password := flag.String("password", "", "Paprika 3 password")
	flag.Parse()

	if *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "username and password are required")
		os.Exit(1)
	}

	s := server.NewMCPServer("paprika-3-mcp", "1.0.0", server.WithLogging(), server.WithResourceCapabilities(false, false))
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	paprika3, err := paprika.NewClient(*username, *password, logger)
	if err != nil {
		slog.Error("failed to create paprika client", "error", err)
		os.Exit(1)
	}

	listRecipesResource := mcp.NewResource("paprika://recipes", "Paprika 3 Recipes", mcp.WithResourceDescription("All recipes in Paprika 3 Recipe Manager"))
	getRecipeResource := mcp.NewResource("paprika://recipes/{uid}", "Paprika 3 Recipe", mcp.WithResourceDescription("An individual recipe in Paprika 3 Recipe Manager"))

	s.AddResource(listRecipesResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		recipes, err := paprika3.ListRecipes(ctx)
		if err != nil {
			return nil, err
		}

		resourceContents := []mcp.ResourceContents{}
		for _, r := range recipes.Result {
			recipe, err := paprika3.GetRecipe(ctx, r.UID)
			if err != nil {
				return nil, err
			}

			if recipe.InTrash {
				continue
			}

			jsonString, err := json.Marshal(recipe)
			if err != nil {
				return nil, err
			}
			resourceContents = append(resourceContents, mcp.TextResourceContents{
				URI:      fmt.Sprintf("paprika://recipes/%s", r.UID),
				MIMEType: "application/json",
				Text:     string(jsonString),
			})
		}

		return resourceContents, nil
	})

	s.AddResource(getRecipeResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		uid, err := extractUID(request.Params.URI)
		if err != nil {
			return nil, err
		}

		recipe, err := paprika3.GetRecipe(ctx, uid)
		if err != nil {
			return nil, err
		}

		jsonString, err := json.Marshal(recipe)
		if err != nil {
			return nil, err
		}
		resourceContents := mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonString),
		}
		return []mcp.ResourceContents{resourceContents}, nil
	})

	createRecipeTool := mcp.NewTool("paprika_create_recipe",
		mcp.WithDescription("Create & Update recipes in the Paprika 3 app"),
		mcp.WithString("name", mcp.Description("The name of the recipe"), mcp.Required()),
		mcp.WithString("ingredients", mcp.Description("The ingredients of the recipe"), mcp.Required()),
		mcp.WithString("directions", mcp.Description("The directions for the recipe"), mcp.Required()),
		mcp.WithString("servings", mcp.Description("The number of servings for the recipe")),
		mcp.WithString("prep_time", mcp.Description("The prep time for the recipe")),
		mcp.WithString("cook_time", mcp.Description("The cook time for the recipe")),
	)

	s.AddTool(createRecipeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, ok := req.Params.Arguments["name"].(string)
		if !ok || len(name) == 0 {
			return nil, errors.New("name is required")
		}
		ingredients, ok := req.Params.Arguments["ingredients"].(string)
		if !ok || len(ingredients) == 0 {
			return nil, errors.New("ingredients are required")
		}
		directions, ok := req.Params.Arguments["directions"].(string)
		if !ok || len(directions) == 0 {
			return nil, errors.New("directions are required")
		}
		servings := req.Params.Arguments["servings"].(string)
		prepTime := req.Params.Arguments["prep_time"].(string)
		cookTime := req.Params.Arguments["cook_time"].(string)

		recipe, err := paprika3.CreateRecipe(ctx, paprika.Recipe{
			Name:        name,
			Ingredients: ingredients,
			Directions:  directions,
			Servings:    servings,
			PrepTime:    prepTime,
			CookTime:    cookTime,
		})
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultResource(recipe.Name, mcp.TextResourceContents{
			URI:      fmt.Sprintf("paprika://recipes/%s", recipe.UID),
			MIMEType: "application/json",
		}), nil
	})

	if err := server.ServeStdio(s); err != nil {
		slog.Error("Server error", "error", err)
	}
}
