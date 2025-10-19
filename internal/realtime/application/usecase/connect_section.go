package usecase

import (
	"context"
	"errors"
	"log"
	"strings"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/domain"
	"mesaYaWs/internal/shared/auth"
)

type ConnectSectionInput struct {
	Token     string
	SectionID string
}

type ConnectSectionOutput struct {
	Claims   *auth.Claims
	Snapshot *domain.SectionSnapshot
}

type ConnectSectionUseCase struct {
	Validator       auth.TokenValidator
	SnapshotFetcher port.SectionSnapshotFetcher
}

var (
	ErrMissingToken   = errors.New("missing token")
	ErrMissingSection = errors.New("missing section id")
)

func NewConnectSectionUseCase(validator auth.TokenValidator, fetcher port.SectionSnapshotFetcher) *ConnectSectionUseCase {
	return &ConnectSectionUseCase{
		Validator:       validator,
		SnapshotFetcher: fetcher,
	}
}

func (uc *ConnectSectionUseCase) Execute(ctx context.Context, input ConnectSectionInput) (*ConnectSectionOutput, error) {
	if strings.TrimSpace(input.Token) == "" {
		return nil, ErrMissingToken
	}
	if strings.TrimSpace(input.SectionID) == "" {
		return nil, ErrMissingSection
	}

	log.Printf("connect-section: validating token section=%s", input.SectionID)

	claims, err := uc.Validator.Validate(input.Token)
	if err != nil {
		log.Printf("connect-section: token validation failed section=%s err=%v", input.SectionID, err)
		return nil, err
	}
	log.Printf("connect-section: token valid section=%s subject=%s session=%s roles=%v", input.SectionID, claims.RegisteredClaims.Subject, claims.SessionID, claims.Roles)

	snapshot, err := uc.SnapshotFetcher.FetchSection(ctx, input.Token, input.SectionID)
	if errors.Is(err, port.ErrSnapshotNotFound) {
		snapshot = nil
		log.Printf("connect-section: snapshot not found section=%s", input.SectionID)
		err = nil
	}
	if err != nil {
		log.Printf("connect-section: snapshot fetch failed section=%s err=%v", input.SectionID, err)
		return nil, err
	}
	if snapshot != nil {
		log.Printf("connect-section: snapshot fetched section=%s payloadType=%T", input.SectionID, snapshot.Payload)
	}

	return &ConnectSectionOutput{Claims: claims, Snapshot: snapshot}, nil
}
