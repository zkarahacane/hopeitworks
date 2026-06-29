package git

import (
	"context"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// reconcilingProvider wraps a GitProvider so every operation error is fed back to the
// credential resolver, which self-heals the advisory connection status on a
// definitive auth failure (C1): 401 -> invalid, 403 -> insufficient_scope. Transient
// failures (429/5xx) are classified as no-ops by the resolver. This is the single
// seam covering git_branch/git_pr/git_merge/ci_poll/hitl_gate without editing each
// action — the decorator sits between the action and the real provider.
type reconcilingProvider struct {
	inner     port.GitProvider
	resolver  port.GitCredentialResolver
	projectID uuid.UUID
}

func newReconcilingProvider(inner port.GitProvider, resolver port.GitCredentialResolver, projectID uuid.UUID) port.GitProvider {
	if resolver == nil {
		return inner
	}
	return &reconcilingProvider{inner: inner, resolver: resolver, projectID: projectID}
}

func (p *reconcilingProvider) note(ctx context.Context, err error) error {
	if err != nil {
		p.resolver.ReconcileFromOperationError(ctx, p.projectID, err)
	}
	return err
}

func (p *reconcilingProvider) CloneRepo(ctx context.Context, repoURL, targetDir string) error {
	return p.note(ctx, p.inner.CloneRepo(ctx, repoURL, targetDir))
}

func (p *reconcilingProvider) CreateBranch(ctx context.Context, workDir, branchName string) error {
	return p.note(ctx, p.inner.CreateBranch(ctx, workDir, branchName))
}

func (p *reconcilingProvider) Push(ctx context.Context, workDir, commitMsg string) error {
	return p.note(ctx, p.inner.Push(ctx, workDir, commitMsg))
}

func (p *reconcilingProvider) CreatePR(ctx context.Context, workDir, title, body, baseBranch string) (string, error) {
	url, err := p.inner.CreatePR(ctx, workDir, title, body, baseBranch)
	return url, p.note(ctx, err)
}

func (p *reconcilingProvider) MergePR(ctx context.Context, workDir, prIdentifier string) error {
	return p.note(ctx, p.inner.MergePR(ctx, workDir, prIdentifier))
}

func (p *reconcilingProvider) GetCIStatus(ctx context.Context, workDir string) (string, error) {
	status, err := p.inner.GetCIStatus(ctx, workDir)
	return status, p.note(ctx, err)
}

func (p *reconcilingProvider) GetPRDiff(ctx context.Context, prURL string) (string, error) {
	diff, err := p.inner.GetPRDiff(ctx, prURL)
	return diff, p.note(ctx, err)
}

func (p *reconcilingProvider) CreateRemoteBranch(ctx context.Context, repoURL, branchName, baseBranch string) error {
	return p.note(ctx, p.inner.CreateRemoteBranch(ctx, repoURL, branchName, baseBranch))
}

func (p *reconcilingProvider) CreateRemotePR(ctx context.Context, repoURL, title, body, headBranch, baseBranch string) (string, error) {
	url, err := p.inner.CreateRemotePR(ctx, repoURL, title, body, headBranch, baseBranch)
	return url, p.note(ctx, err)
}

func (p *reconcilingProvider) GetRemoteCIStatus(ctx context.Context, prURL string) (string, error) {
	status, err := p.inner.GetRemoteCIStatus(ctx, prURL)
	return status, p.note(ctx, err)
}
