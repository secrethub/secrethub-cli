package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-go/internals/api"
)

func getAuditActor(event api.Audit) (string, error) {
	if event.Actor.Deleted {
		return event.Actor.ActorID.String(), nil
	}

	switch event.Actor.Type {
	case "user":
		return event.Actor.User.Username, nil
	case "service":
		return event.Actor.Service.ServiceID, nil
	}

	return "", ErrInvalidAuditActor
}

func getAuditSubject(event api.Audit, tree *api.Tree) (string, error) {
	if event.Subject.Deleted {
		return event.Subject.SubjectID.String(), nil
	}

	switch event.Subject.Type {
	case api.AuditSubjectUser:
		return event.Subject.User.PrettyName(), nil
	case api.AuditSubjectService:
		return event.Subject.Service.ServiceID, nil
	case api.AuditSubjectRepo:
		return event.Subject.Repo.Name, nil
	case api.AuditSubjectRepoKey:
		return event.Subject.Repo.Name, nil
	case api.AuditSubjectSecret:
		secretPath, err := tree.AbsSecretPath(event.Subject.Secret.SecretID)
		if err != nil {
			return "", err
		}
		return secretPath.String(), nil
	case api.AuditSubjectSecretVersion:
		secretPath, err := tree.AbsSecretPath(event.Subject.SecretVersion.Secret.SecretID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s:%d", secretPath.String(), event.Subject.SecretVersion.Version), nil
	case api.AuditSubjectSecretKey:
		return event.Subject.SubjectID.ToString(), nil
	case api.AuditSubjectSecretMember:
		account := ""
		if event.Subject.User != nil {
			account = event.Subject.User.Username
		} else if event.Subject.Service != nil {
			account = event.Subject.Service.ServiceID
		} else {
			return "", ErrInvalidAuditSubject
		}
		return fmt.Sprintf("%s => %s", account, event.Subject.Secret.Name), nil
	}

	return "", ErrInvalidAuditSubject

}

func getEventAction(event api.Audit) string {
	action := event.Action
	subjectType := event.Subject.Type

	if event.Subject.Type == api.AuditSubjectUser {
		if event.Action == api.AuditActionCreate {
			action = "invite"
		} else if event.Action == api.AuditActionDelete {
			action = "revoke"
		}
	}

	return fmt.Sprintf("%s.%s", action, subjectType)

}
