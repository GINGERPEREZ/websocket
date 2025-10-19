package usecase

import (
	"context"
	"errors"
	"log"
	"net/url"
	"strings"
	"time"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/domain"
	"mesaYaWs/internal/shared/auth"
)

type ConnectSectionInput struct {
	Token     string
	SectionID string
	Query     url.Values
}

type ConnectSectionOutput struct {
	Claims   *auth.Claims
	Snapshot *domain.SectionSnapshot
}

type ConnectSectionUseCase struct {
	Validator       auth.TokenValidator
	SnapshotFetcher port.SectionSnapshotFetcher
	cache           *snapshotCache
}

var (
	ErrMissingToken   = errors.New("missing token")
	ErrMissingSection = errors.New("missing section id")
)

func NewConnectSectionUseCase(validator auth.TokenValidator, fetcher port.SectionSnapshotFetcher) *ConnectSectionUseCase {
	return &ConnectSectionUseCase{
		Validator:       validator,
		SnapshotFetcher: fetcher,
		cache:           newSnapshotCache(),
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

	normalizedQuery := normalizeQueryParams(input.Query)
	queryKey := canonicalQuery(normalizedQuery)
	if queryKey != "" {
		log.Printf("connect-section: snapshot params section=%s query=%s", input.SectionID, queryKey)
	}

	fetchQuery := cloneValues(normalizedQuery)
	snapshot, err := uc.SnapshotFetcher.FetchSection(ctx, input.Token, input.SectionID, fetchQuery)
	switch {
	case errors.Is(err, port.ErrSnapshotNotFound):
		log.Printf("connect-section: snapshot not found section=%s", input.SectionID)
		uc.cache.delete(input.SectionID, normalizedQuery)
		snapshot = nil
		err = nil
	case err != nil:
		log.Printf("connect-section: snapshot fetch failed section=%s query=%s err=%v", input.SectionID, queryKey, err)
		if cached, ok := uc.cache.get(input.SectionID, normalizedQuery); ok && cached.snapshot != nil {
			log.Printf("connect-section: using cached snapshot section=%s query=%s fetchedAt=%s", input.SectionID, queryKey, cached.fetchedAt.Format(time.RFC3339Nano))
			uc.cache.set(input.SectionID, normalizedQuery, input.Token, cached.snapshot)
			return &ConnectSectionOutput{Claims: claims, Snapshot: cached.snapshot}, nil
		}
		return nil, err
	default:
		uc.cache.set(input.SectionID, normalizedQuery, input.Token, snapshot)
	}

	if snapshot != nil {
		log.Printf("connect-section: snapshot fetched section=%s payloadType=%T", input.SectionID, snapshot.Payload)
	}

	return &ConnectSectionOutput{Claims: claims, Snapshot: snapshot}, nil
}

func (uc *ConnectSectionUseCase) RefreshSectionSnapshots(ctx context.Context, entity, sectionID string, broadcaster *BroadcastUseCase) {
	entries := uc.cache.entriesForSection(sectionID)
	if len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		queryKey := canonicalQuery(entry.query)
		snapshot, err := uc.SnapshotFetcher.FetchSection(ctx, entry.token, sectionID, cloneValues(entry.query))
		switch {
		case errors.Is(err, port.ErrSnapshotNotFound):
			log.Printf("connect-section: refresh snapshot not found section=%s query=%s", sectionID, queryKey)
			uc.cache.delete(sectionID, entry.query)
			continue
		case err != nil:
			log.Printf("connect-section: refresh snapshot failed section=%s query=%s err=%v", sectionID, queryKey, err)
			continue
		default:
			uc.cache.set(sectionID, entry.query, entry.token, snapshot)
		}

		metadata := map[string]string{
			"sectionId": sectionID,
		}
		for key, values := range entry.query {
			if len(values) == 0 {
				continue
			}
			metadata["query."+strings.ToLower(strings.TrimSpace(key))] = values[0]
		}

		message := &domain.Message{
			Topic:      entity + ".snapshot",
			Entity:     entity,
			Action:     "snapshot",
			ResourceID: sectionID,
			Metadata:   metadata,
			Data:       snapshot.Payload,
			Timestamp:  time.Now().UTC(),
		}
		broadcaster.Execute(ctx, message)
		log.Printf("connect-section: refreshed snapshot broadcast section=%s entity=%s query=%s", sectionID, entity, queryKey)
	}
}

func normalizeQueryParams(raw url.Values) url.Values {
	if raw == nil {
		raw = url.Values{}
	}
	normalized := url.Values{}
	for key, values := range raw {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || strings.EqualFold(trimmedKey, "token") {
			continue
		}
		for _, value := range values {
			trimmedValue := strings.TrimSpace(value)
			if trimmedValue == "" {
				continue
			}
			normalized.Set(trimmedKey, trimmedValue)
		}
	}
	if normalized.Get("page") == "" {
		normalized.Set("page", "1")
	}
	if normalized.Get("limit") == "" {
		normalized.Set("limit", "20")
	}
	return normalized
}
