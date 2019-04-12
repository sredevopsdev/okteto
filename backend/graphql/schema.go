package graphql

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/okteto/app/backend/app"
	"github.com/okteto/app/backend/github"
	"github.com/okteto/app/backend/log"
	"github.com/okteto/app/backend/model"
)

type credential struct {
	Config string
}

var devEnvironmentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DevEnvironment",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.ID,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"endpoints": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
		},
	},
)

var credentialsType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Credential",
		Fields: graphql.Fields{
			"config": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

var authenticatedUserType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "me",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.ID,
			},
			"email": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"token": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"environments": &graphql.Field{
				Type:        graphql.NewList(devEnvironmentType),
				Description: "Get environment list",
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					u, err := validateToken(params.Context)
					if err != nil {
						return nil, err
					}

					l, err := app.ListDevEnvs(u)
					if err != nil {
						log.Errorf("failed to get dev envs for %s", u)
						return nil, fmt.Errorf("failed to get your environments")
					}

					return l, nil
				},
			},
			"credentials": &graphql.Field{
				Type:        credentialsType,
				Description: "Get credentials of the space",
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					u, err := validateToken(params.Context)
					if err != nil {
						return nil, err
					}

					c, err := app.GetCredential(u)
					if err != nil {
						log.Errorf("failed to get credentials: %s", err)
						return nil, fmt.Errorf("failed to get credentials")
					}

					return credential{Config: c}, nil
				},
			},
		},
	})

var mutationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"auth": &graphql.Field{
				Type:        authenticatedUserType,
				Description: "Authenticate a user with github",
				Args: graphql.FieldConfigArgument{
					"code": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {

					code := params.Args["code"].(string)
					u, err := github.Auth(code)
					if err != nil {
						log.Errorf("failed to auth user: %s", err)
						return nil, fmt.Errorf("failed to authenticate")
					}

					if _, err := app.CreateSpace(u.ID); err != nil {
						log.Errorf("failed to create space for %s: %s", u.ID, err)
						return nil, fmt.Errorf("failed to create your space")
					}

					return u, nil
				},
			},
			"up": &graphql.Field{
				Type:        devEnvironmentType,
				Description: "Create dev mode",
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"image": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"workdir": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"services": &graphql.ArgumentConfig{
						Type: graphql.NewList(graphql.String),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					u, err := validateToken(params.Context)
					if err != nil {
						return nil, err
					}

					d := buildDev(params.Args)
					s := &model.Space{
						Name: u,
					}

					if err := app.DevModeOn(d, s); err != nil {
						log.Errorf("failed to enable dev mode: %s", err)
						return nil, fmt.Errorf("failed to enable dev mode")
					}

					return d, nil

				},
			},
			"down": &graphql.Field{
				Type:        devEnvironmentType,
				Description: "Delete dev space",
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					u, err := validateToken(params.Context)
					if err != nil {
						return nil, err
					}

					d := &model.Dev{
						Name: params.Args["name"].(string),
					}

					s := &model.Space{
						Name: u,
					}

					if err := app.DevModeOff(d, s, false); err != nil {
						log.Errorf("failed to enable dev mode: %s", err)
						return nil, fmt.Errorf("failed to enable dev mode")
					}

					return d, nil

				},
			},
		},
	},
)

// Schema holds the GraphQL schema for Okteto
var Schema, _ = graphql.NewSchema(
	graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	},
)

func buildDev(args map[string]interface{}) *model.Dev {
	aInterface := args["services"].([]interface{})
	sv := make([]string, len(aInterface))
	for i, v := range aInterface {
		sv[i] = v.(string)
	}

	d := &model.Dev{
		Name:     args["name"].(string),
		Image:    args["image"].(string),
		WorkDir:  args["workdir"].(string),
		Services: sv,
	}

	return d
}
