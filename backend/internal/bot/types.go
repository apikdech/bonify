package bot

// Update represents a Telegram webhook update
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message represents a Telegram message
type Message struct {
	MessageID int64       `json:"message_id"`
	From      *User       `json:"from,omitempty"`
	Chat      *Chat       `json:"chat"`
	Date      int64       `json:"date"`
	Text      string      `json:"text,omitempty"`
	Photo     []PhotoSize `json:"photo,omitempty"`
}

// User represents a Telegram user
type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// PhotoSize represents a photo in various sizes
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

// ============================================
// Discord API Types
// ============================================

// InteractionType represents the type of Discord interaction
type InteractionType int

const (
	InteractionTypePing                           InteractionType = 1
	InteractionTypeApplicationCommand             InteractionType = 2
	InteractionTypeMessageComponent               InteractionType = 3
	InteractionTypeApplicationCommandAutocomplete InteractionType = 4
	InteractionTypeModalSubmit                    InteractionType = 5
)

// Interaction represents a Discord interaction (slash command, etc.)
type Interaction struct {
	ID            string           `json:"id"`
	ApplicationID string           `json:"application_id"`
	Type          InteractionType  `json:"type"`
	Data          *InteractionData `json:"data,omitempty"`
	GuildID       string           `json:"guild_id,omitempty"`
	ChannelID     string           `json:"channel_id,omitempty"`
	Member        *GuildMember     `json:"member,omitempty"`
	User          *DiscordUser     `json:"user,omitempty"`
	Token         string           `json:"token"`
	Version       int              `json:"version"`
	Message       *DiscordMessage  `json:"message,omitempty"`
	Locale        string           `json:"locale,omitempty"`
	GuildLocale   string           `json:"guild_locale,omitempty"`
}

// InteractionData represents the data payload of an interaction
type InteractionData struct {
	ID       string                     `json:"id"`
	Name     string                     `json:"name"`
	Type     int                        `json:"type"`
	Resolved *ResolvedData              `json:"resolved,omitempty"`
	Options  []ApplicationCommandOption `json:"options,omitempty"`
	GuildID  string                     `json:"guild_id,omitempty"`
	TargetID string                     `json:"target_id,omitempty"`
}

// ResolvedData contains resolved data for users, members, roles, channels, and attachments
type ResolvedData struct {
	Users       map[string]DiscordUser `json:"users,omitempty"`
	Members     map[string]GuildMember `json:"members,omitempty"`
	Roles       map[string]interface{} `json:"roles,omitempty"`
	Channels    map[string]interface{} `json:"channels,omitempty"`
	Attachments map[string]Attachment  `json:"attachments,omitempty"`
}

// GuildMember represents a member of a Discord guild (server)
type GuildMember struct {
	User                       *DiscordUser `json:"user,omitempty"`
	Nick                       string       `json:"nick,omitempty"`
	Avatar                     string       `json:"avatar,omitempty"`
	Roles                      []string     `json:"roles,omitempty"`
	JoinedAt                   string       `json:"joined_at,omitempty"`
	PremiumSince               string       `json:"premium_since,omitempty"`
	Deaf                       bool         `json:"deaf"`
	Mute                       bool         `json:"mute"`
	Flags                      int          `json:"flags,omitempty"`
	Pending                    bool         `json:"pending,omitempty"`
	Permissions                string       `json:"permissions,omitempty"`
	CommunicationDisabledUntil string       `json:"communication_disabled_until,omitempty"`
}

// DiscordUser represents a Discord user
type DiscordUser struct {
	ID                   string      `json:"id"`
	Username             string      `json:"username"`
	Discriminator        string      `json:"discriminator,omitempty"`
	GlobalName           string      `json:"global_name,omitempty"`
	Avatar               string      `json:"avatar,omitempty"`
	Bot                  bool        `json:"bot,omitempty"`
	System               bool        `json:"system,omitempty"`
	MFAEnabled           bool        `json:"mfa_enabled,omitempty"`
	Banner               string      `json:"banner,omitempty"`
	AccentColor          int         `json:"accent_color,omitempty"`
	Locale               string      `json:"locale,omitempty"`
	Verified             bool        `json:"verified,omitempty"`
	Email                string      `json:"email,omitempty"`
	Flags                int         `json:"flags,omitempty"`
	PremiumType          int         `json:"premium_type,omitempty"`
	PublicFlags          int         `json:"public_flags,omitempty"`
	AvatarDecorationData interface{} `json:"avatar_decoration_data,omitempty"`
}

// DiscordMessage represents a Discord message (simplified)
type DiscordMessage struct {
	ID              string       `json:"id"`
	ChannelID       string       `json:"channel_id"`
	Author          *DiscordUser `json:"author,omitempty"`
	Content         string       `json:"content,omitempty"`
	Timestamp       string       `json:"timestamp"`
	EditedTimestamp string       `json:"edited_timestamp,omitempty"`
	TTS             bool         `json:"tts,omitempty"`
	MentionEveryone bool         `json:"mention_everyone,omitempty"`
	Attachments     []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a Discord message attachment
type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Description string `json:"description,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url"`
	Height      int    `json:"height,omitempty"`
	Width       int    `json:"width,omitempty"`
	Ephemeral   bool   `json:"ephemeral,omitempty"`
}

// InteractionResponseType represents the type of interaction response
type InteractionResponseType int

const (
	InteractionResponseTypePong                                 InteractionResponseType = 1
	InteractionResponseTypeChannelMessageWithSource             InteractionResponseType = 4
	InteractionResponseTypeDeferredChannelMessageWithSource     InteractionResponseType = 5
	InteractionResponseTypeDeferredUpdateMessage                InteractionResponseType = 6
	InteractionResponseTypeUpdateMessage                        InteractionResponseType = 7
	InteractionResponseTypeApplicationCommandAutocompleteResult InteractionResponseType = 8
	InteractionResponseTypeModal                                InteractionResponseType = 9
)

// InteractionResponse represents a response to a Discord interaction
type InteractionResponse struct {
	Type InteractionResponseType  `json:"type"`
	Data *InteractionResponseData `json:"data,omitempty"`
}

// InteractionResponseData represents the data payload of an interaction response
type InteractionResponseData struct {
	TTS             bool          `json:"tts,omitempty"`
	Content         string        `json:"content,omitempty"`
	Embeds          []interface{} `json:"embeds,omitempty"`
	AllowedMentions *interface{}  `json:"allowed_mentions,omitempty"`
	Flags           int           `json:"flags,omitempty"`
	Components      []interface{} `json:"components,omitempty"`
	Attachments     []interface{} `json:"attachments,omitempty"`
}

// InteractionResponseFlags for response flags
const (
	InteractionResponseFlagsEphemeral = 64 // Message is only visible to the user
)

// ApplicationCommandOptionType represents the type of command option
type ApplicationCommandOptionType int

const (
	ApplicationCommandOptionTypeSubCommand      ApplicationCommandOptionType = 1
	ApplicationCommandOptionTypeSubCommandGroup ApplicationCommandOptionType = 2
	ApplicationCommandOptionTypeString          ApplicationCommandOptionType = 3
	ApplicationCommandOptionTypeInteger         ApplicationCommandOptionType = 4
	ApplicationCommandOptionTypeBoolean         ApplicationCommandOptionType = 5
	ApplicationCommandOptionTypeUser            ApplicationCommandOptionType = 6
	ApplicationCommandOptionTypeChannel         ApplicationCommandOptionType = 7
	ApplicationCommandOptionTypeRole            ApplicationCommandOptionType = 8
	ApplicationCommandOptionTypeMentionable     ApplicationCommandOptionType = 9
	ApplicationCommandOptionTypeNumber          ApplicationCommandOptionType = 10
	ApplicationCommandOptionTypeAttachment      ApplicationCommandOptionType = 11
)

// ApplicationCommandOption represents an option for a slash command
type ApplicationCommandOption struct {
	Name         string                       `json:"name"`
	Type         ApplicationCommandOptionType `json:"type"`
	Value        interface{}                  `json:"value,omitempty"`
	Description  string                       `json:"description,omitempty"`
	Required     bool                         `json:"required,omitempty"`
	Options      []ApplicationCommandOption   `json:"options,omitempty"`
	Choices      []interface{}                `json:"choices,omitempty"`
	ChannelTypes []int                        `json:"channel_types,omitempty"`
	MinValue     float64                      `json:"min_value,omitempty"`
	MaxValue     float64                      `json:"max_value,omitempty"`
	MinLength    int                          `json:"min_length,omitempty"`
	MaxLength    int                          `json:"max_length,omitempty"`
	Autocomplete bool                         `json:"autocomplete,omitempty"`
}
