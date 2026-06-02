package namespace

import "time"

type IdentityGroupCache struct {
	ID              string
	GroupIdentifier string
	ResolvedMembers []string
	LastRefreshedAt time.Time
	TTLSeconds      int
}

func (c *IdentityGroupCache) IsExpired() bool {
	return time.Since(c.LastRefreshedAt) > time.Duration(c.TTLSeconds)*time.Second
}

func (c *IdentityGroupCache) ContainsUser(username string) bool {
	for _, m := range c.ResolvedMembers {
		if m == username {
			return true
		}
	}
	return false
}
