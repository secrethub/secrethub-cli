package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestAddTreeToPlan(t *testing.T) {
	uuids := make([]uuid.UUID, 10)
	for i := 0; i < len(uuids); i++ {
		uuids[i] = uuid.New()
	}

	cases := map[string]struct {
		tree     *api.Tree
		err      error
		expected *plan
	}{
		"single project with flat secrets": {
			tree: createTree(
				&api.Dir{
					DirID:    uuids[0],
					ParentID: nil,
					Name:     "my-project",
					SubDirs:  []*api.Dir{},
					Secrets: []*api.Secret{
						{
							Name:     "stripe-api-key",
							DirID:    uuids[0],
							SecretID: uuid.New(),
						},
						{
							Name:     "aws-access-key-id",
							DirID:    uuids[0],
							SecretID: uuid.New(),
						},
						{
							Name:     "aws-secret-access-key",
							DirID:    uuids[0],
							SecretID: uuid.New(),
						},
						{
							Name:     "db-user",
							DirID:    uuids[0],
							SecretID: uuid.New(),
						},
						{
							Name:     "db-password",
							DirID:    uuids[0],
							SecretID: uuid.New(),
						},
					},
				},
				"company",
			),
			err: nil,
			expected: &plan{
				vaults: map[string]*vault{
					"my-project": {
						Name: "my-project",
						Items: []item{
							{
								Name: "stripe-api-key",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/stripe-api-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws-access-key-id",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/aws-access-key-id",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws-secret-access-key",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/aws-secret-access-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "db-user",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/db-user",
										Concealed: true,
									},
								},
							},
							{
								Name: "db-password",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/db-password",
										Concealed: true,
									},
								},
							},
						},
					},
				},
			},
		},
		"single project with related secrets grouped in dirs": {
			tree: createTree(
				&api.Dir{
					DirID:    uuids[0],
					ParentID: nil,
					Name:     "my-project",
					SubDirs: []*api.Dir{
						{
							Name:     "stripe",
							DirID:    uuids[1],
							ParentID: &uuids[0],
							SubDirs:  []*api.Dir{},
							Secrets: []*api.Secret{
								{
									Name:     "api-key",
									DirID:    uuids[1],
									SecretID: uuid.New(),
								},
							},
						},
						{
							Name:     "aws",
							DirID:    uuids[2],
							ParentID: &uuids[0],
							SubDirs:  []*api.Dir{},
							Secrets: []*api.Secret{
								{
									Name:     "access-key-id",
									DirID:    uuids[2],
									SecretID: uuid.New(),
								},
								{
									Name:     "secret-access-key",
									DirID:    uuids[2],
									SecretID: uuid.New(),
								},
							},
						},
						{
							Name:     "db",
							DirID:    uuids[3],
							ParentID: &uuids[0],
							SubDirs:  []*api.Dir{},
							Secrets: []*api.Secret{
								{
									Name:     "user",
									DirID:    uuids[3],
									SecretID: uuid.New(),
								},
								{
									Name:     "password",
									DirID:    uuids[3],
									SecretID: uuid.New(),
								},
							},
						},
					},
					Secrets: []*api.Secret{},
				},
				"company",
			),
			err: nil,
			expected: &plan{
				vaults: map[string]*vault{
					"my-project": {
						Name: "my-project",
						Items: []item{
							{
								Name: "stripe",
								Fields: []field{
									{
										Name:      "api-key",
										Reference: "secrethub://company/my-project/stripe/api-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "access-key-id",
										Reference: "secrethub://company/my-project/aws/access-key-id",
										Concealed: false,
									},
									{
										Name:      "secret-access-key",
										Reference: "secrethub://company/my-project/aws/secret-access-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "db",
								Fields: []field{
									{
										Name:      "user",
										Reference: "secrethub://company/my-project/db/user",
										Concealed: false,
									},
									{
										Name:      "password",
										Reference: "secrethub://company/my-project/db/password",
										Concealed: true,
									},
								},
							},
						},
					},
				},
			},
		},
		"multiple environments": {
			tree: createTree(
				&api.Dir{
					DirID:    uuids[0],
					ParentID: nil,
					Name:     "my-project",
					SubDirs: []*api.Dir{
						{
							Name:     "dev",
							DirID:    uuids[1],
							ParentID: &uuids[0],
							SubDirs: []*api.Dir{
								{
									Name:     "stripe",
									DirID:    uuids[2],
									ParentID: &uuids[1],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "api-key",
											DirID:    uuids[2],
											SecretID: uuid.New(),
										},
									},
								},
								{
									Name:     "aws",
									DirID:    uuids[3],
									ParentID: &uuids[1],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "access-key-id",
											DirID:    uuids[3],
											SecretID: uuid.New(),
										},
										{
											Name:     "secret-access-key",
											DirID:    uuids[3],
											SecretID: uuid.New(),
										},
									},
								},
								{
									Name:     "db",
									DirID:    uuids[4],
									ParentID: &uuids[1],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "user",
											DirID:    uuids[4],
											SecretID: uuid.New(),
										},
										{
											Name:     "password",
											DirID:    uuids[4],
											SecretID: uuid.New(),
										},
									},
								},
							},
							Secrets: []*api.Secret{},
						},
						{
							Name:     "prd",
							DirID:    uuids[5],
							ParentID: &uuids[0],
							SubDirs: []*api.Dir{
								{
									Name:     "stripe",
									DirID:    uuids[6],
									ParentID: &uuids[5],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "api-key",
											DirID:    uuids[6],
											SecretID: uuid.New(),
										},
									},
								},
								{
									Name:     "aws",
									DirID:    uuids[7],
									ParentID: &uuids[5],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "access-key-id",
											DirID:    uuids[7],
											SecretID: uuid.New(),
										},
										{
											Name:     "secret-access-key",
											DirID:    uuids[7],
											SecretID: uuid.New(),
										},
									},
								},
								{
									Name:     "db",
									DirID:    uuids[8],
									ParentID: &uuids[5],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "user",
											DirID:    uuids[8],
											SecretID: uuid.New(),
										},
										{
											Name:     "password",
											DirID:    uuids[8],
											SecretID: uuid.New(),
										},
									},
								},
							},
							Secrets: []*api.Secret{},
						},
					},
					Secrets: []*api.Secret{},
				},
				"company",
			),
			err: nil,
			expected: &plan{
				vaults: map[string]*vault{
					"my-project-dev": {
						Name: "my-project-dev",
						Items: []item{
							{
								Name: "stripe",
								Fields: []field{
									{
										Name:      "api-key",
										Reference: "secrethub://company/my-project/dev/stripe/api-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "access-key-id",
										Reference: "secrethub://company/my-project/dev/aws/access-key-id",
										Concealed: false,
									},
									{
										Name:      "secret-access-key",
										Reference: "secrethub://company/my-project/dev/aws/secret-access-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "db",
								Fields: []field{
									{
										Name:      "user",
										Reference: "secrethub://company/my-project/dev/db/user",
										Concealed: false,
									},
									{
										Name:      "password",
										Reference: "secrethub://company/my-project/dev/db/password",
										Concealed: true,
									},
								},
							},
						},
					},
					"my-project-prd": {
						Name: "my-project-prd",
						Items: []item{
							{
								Name: "stripe",
								Fields: []field{
									{
										Name:      "api-key",
										Reference: "secrethub://company/my-project/prd/stripe/api-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "access-key-id",
										Reference: "secrethub://company/my-project/prd/aws/access-key-id",
										Concealed: false,
									},
									{
										Name:      "secret-access-key",
										Reference: "secrethub://company/my-project/prd/aws/secret-access-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "db",
								Fields: []field{
									{
										Name:      "user",
										Reference: "secrethub://company/my-project/prd/db/user",
										Concealed: false,
									},
									{
										Name:      "password",
										Reference: "secrethub://company/my-project/prd/db/password",
										Concealed: true,
									},
								},
							},
						},
					},
				},
			},
		},
		"mixed grouped secrets and flat secrets": {
			tree: createTree(
				&api.Dir{
					DirID:    uuids[0],
					ParentID: nil,
					Name:     "my-project",
					SubDirs: []*api.Dir{
						{
							Name:     "db",
							DirID:    uuids[1],
							ParentID: &uuids[0],
							SubDirs:  []*api.Dir{},
							Secrets: []*api.Secret{
								{
									Name:     "user",
									DirID:    uuids[1],
									SecretID: uuid.New(),
								},
								{
									Name:     "password",
									DirID:    uuids[1],
									SecretID: uuid.New(),
								},
							},
						},
					},
					Secrets: []*api.Secret{
						{
							Name:     "stripe-api-key",
							DirID:    uuids[0],
							SecretID: uuid.New(),
						},
					},
				},
				"company",
			),
			err: nil,
			expected: &plan{
				vaults: map[string]*vault{
					"my-project": {
						Name: "my-project",
						Items: []item{
							{
								Name: "stripe-api-key",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/stripe-api-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "db",
								Fields: []field{
									{
										Name:      "user",
										Reference: "secrethub://company/my-project/db/user",
										Concealed: false,
									},
									{
										Name:      "password",
										Reference: "secrethub://company/my-project/db/password",
										Concealed: true,
									},
								},
							},
						},
					},
				},
			},
		},
		"underscores in special name": {
			tree: createTree(
				&api.Dir{
					DirID:    uuids[0],
					ParentID: nil,
					Name:     "my-project",
					SubDirs: []*api.Dir{
						{
							Name:     "aws",
							DirID:    uuids[1],
							ParentID: &uuids[0],
							SubDirs:  []*api.Dir{},
							Secrets: []*api.Secret{
								{
									Name:     "secret_access_key",
									DirID:    uuids[1],
									SecretID: uuid.New(),
								},
								{
									Name:     "access_key_id",
									DirID:    uuids[1],
									SecretID: uuid.New(),
								},
							},
						},
					},
					Secrets: []*api.Secret{
						{
							Name:     "stripe_api_key",
							DirID:    uuids[0],
							SecretID: uuid.New(),
						},
					},
				},
				"company",
			),
			err: nil,
			expected: &plan{
				vaults: map[string]*vault{
					"my-project": {
						Name: "my-project",
						Items: []item{
							{
								Name: "stripe_api_key",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/stripe_api_key",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "secret_access_key",
										Reference: "secrethub://company/my-project/aws/secret_access_key",
										Concealed: true,
									},
									{
										Name:      "access_key_id",
										Reference: "secrethub://company/my-project/aws/access_key_id",
										Concealed: false,
									},
								},
							},
						},
					},
				},
			},
		},
		"items and vaults on same tree depth": {
			tree: createTree(
				&api.Dir{
					DirID:    uuids[0],
					ParentID: nil,
					Name:     "my-project",
					SubDirs: []*api.Dir{
						{
							Name:     "dev",
							DirID:    uuids[1],
							ParentID: &uuids[0],
							SubDirs: []*api.Dir{
								{
									Name:     "aws",
									DirID:    uuids[2],
									ParentID: &uuids[1],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "access-key-id",
											DirID:    uuids[2],
											SecretID: uuid.New(),
										},
										{
											Name:     "secret-access-key",
											DirID:    uuids[2],
											SecretID: uuid.New(),
										},
									},
								},
							},
							Secrets: []*api.Secret{},
						},
						{
							Name:     "prd",
							DirID:    uuids[3],
							ParentID: &uuids[0],
							SubDirs: []*api.Dir{
								{
									Name:     "aws",
									DirID:    uuids[4],
									ParentID: &uuids[3],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "access-key-id",
											DirID:    uuids[4],
											SecretID: uuid.New(),
										},
										{
											Name:     "secret-access-key",
											DirID:    uuids[4],
											SecretID: uuid.New(),
										},
									},
								},
							},
							Secrets: []*api.Secret{},
						},
						{
							Name:     "db",
							DirID:    uuids[5],
							ParentID: &uuids[0],
							SubDirs:  []*api.Dir{},
							Secrets: []*api.Secret{
								{
									Name:     "user",
									DirID:    uuids[5],
									SecretID: uuid.New(),
								},
								{
									Name:     "password",
									DirID:    uuids[5],
									SecretID: uuid.New(),
								},
							},
						},
					},
					Secrets: []*api.Secret{},
				},
				"company",
			),
			err: nil,
			expected: &plan{
				vaults: map[string]*vault{
					"my-project-dev": {
						Name: "my-project-dev",
						Items: []item{
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "access-key-id",
										Reference: "secrethub://company/my-project/dev/aws/access-key-id",
										Concealed: false,
									},
									{
										Name:      "secret-access-key",
										Reference: "secrethub://company/my-project/dev/aws/secret-access-key",
										Concealed: true,
									},
								},
							},
						},
					},
					"my-project-prd": {
						Name: "my-project-prd",
						Items: []item{
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "access-key-id",
										Reference: "secrethub://company/my-project/prd/aws/access-key-id",
										Concealed: false,
									},
									{
										Name:      "secret-access-key",
										Reference: "secrethub://company/my-project/prd/aws/secret-access-key",
										Concealed: true,
									},
								},
							},
						},
					},
					"my-project": {
						Name: "my-project",
						Items: []item{
							{
								Name: "db",
								Fields: []field{
									{
										Name:      "user",
										Reference: "secrethub://company/my-project/db/user",
										Concealed: false,
									},
									{
										Name:      "password",
										Reference: "secrethub://company/my-project/db/password",
										Concealed: true,
									},
								},
							},
						},
					},
				},
			},
		},
		"subdir containing a mix of flat and grouped secrets": {
			tree: createTree(
				&api.Dir{
					DirID:    uuids[0],
					ParentID: nil,
					Name:     "my-project",
					SubDirs: []*api.Dir{
						{
							Name:     "dev",
							DirID:    uuids[1],
							ParentID: &uuids[0],
							SubDirs: []*api.Dir{
								{
									Name:     "aws",
									DirID:    uuids[2],
									ParentID: &uuids[1],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "access-key-id",
											DirID:    uuids[2],
											SecretID: uuid.New(),
										},
										{
											Name:     "secret-access-key",
											DirID:    uuids[2],
											SecretID: uuid.New(),
										},
									},
								},
							},
							Secrets: []*api.Secret{
								{
									Name:     "stripe-api-key",
									DirID:    uuids[1],
									SecretID: uuid.New(),
								},
							},
						},
						{
							Name:     "prd",
							DirID:    uuids[3],
							ParentID: &uuids[0],
							SubDirs: []*api.Dir{
								{
									Name:     "aws",
									DirID:    uuids[4],
									ParentID: &uuids[3],
									SubDirs:  []*api.Dir{},
									Secrets: []*api.Secret{
										{
											Name:     "access-key-id",
											DirID:    uuids[4],
											SecretID: uuid.New(),
										},
										{
											Name:     "secret-access-key",
											DirID:    uuids[4],
											SecretID: uuid.New(),
										},
									},
								},
							},
							Secrets: []*api.Secret{
								{
									Name:     "stripe-api-key",
									DirID:    uuids[1],
									SecretID: uuid.New(),
								},
							},
						},
					},
					Secrets: []*api.Secret{},
				},
				"company",
			),
			err: nil,
			expected: &plan{
				vaults: map[string]*vault{
					"my-project-dev": {
						Name: "my-project-dev",
						Items: []item{
							{
								Name: "stripe-api-key",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/dev/stripe-api-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "access-key-id",
										Reference: "secrethub://company/my-project/dev/aws/access-key-id",
										Concealed: false,
									},
									{
										Name:      "secret-access-key",
										Reference: "secrethub://company/my-project/dev/aws/secret-access-key",
										Concealed: true,
									},
								},
							},
						},
					},
					"my-project-prd": {
						Name: "my-project-prd",
						Items: []item{
							{
								Name: "stripe-api-key",
								Fields: []field{
									{
										Name:      "secret",
										Reference: "secrethub://company/my-project/dev/stripe-api-key",
										Concealed: true,
									},
								},
							},
							{
								Name: "aws",
								Fields: []field{
									{
										Name:      "access-key-id",
										Reference: "secrethub://company/my-project/prd/aws/access-key-id",
										Concealed: false,
									},
									{
										Name:      "secret-access-key",
										Reference: "secrethub://company/my-project/prd/aws/secret-access-key",
										Concealed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			plan := newPlan()
			err := addTreeToPlan(tc.tree, plan)

			assert.Equal(t, err, tc.err)
			assert.Equal(t, plan, tc.expected)
		})
	}
}

func createTree(rootDir *api.Dir, parentPath string) *api.Tree {
	tree := &api.Tree{
		RootDir:    rootDir,
		ParentPath: api.ParentPath(parentPath),
		Dirs:       make(map[uuid.UUID]*api.Dir),
		Secrets:    make(map[uuid.UUID]*api.Secret),
	}
	addDirToTree(tree, rootDir)
	return tree
}

func addDirToTree(tree *api.Tree, dir *api.Dir) {
	for _, subDir := range dir.SubDirs {
		tree.Dirs[subDir.DirID] = subDir
		addDirToTree(tree, subDir)
	}
	for _, secret := range dir.Secrets {
		tree.Secrets[secret.SecretID] = secret
	}
}
