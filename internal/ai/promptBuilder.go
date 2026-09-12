package ai

import (
	"fmt"
	"strings"
	"time"

	"mortis/internal/core"
	"mortis/internal/rapport"
)

type Profile interface {
	ToPromptString() string
}

type ProfileContext interface {
	Profile
	FieldNames() []string
}

type PromptBuilder struct {
	modules []core.Module
}

func NewPromptBuilder(modules []core.Module) *PromptBuilder {
	return &PromptBuilder{modules: modules}
}

func (b *PromptBuilder) Build(userInput string, profile Profile, recentRapport []rapport.Entry) string {
	sections := []string{
		persona,
		b.userContext(profile, recentRapport),
		rules,
		outputFormat,
		b.moduleDefinitions(),
		disambiguation,
		failSafe,
	}
	return strings.Join(sections, "\n\n")
}

func (b *PromptBuilder) userContext(profile Profile, recentRapport []rapport.Entry) string {
	profileText := "(no known facts yet)"
	if profile != nil {
		profileText = profile.ToPromptString()
	}

	rapportText := "(none yet)"
	if len(recentRapport) > 0 {
		lines := make([]string, len(recentRapport))
		for i, e := range recentRapport {
			lines[i] = fmt.Sprintf("- [%s] %s", e.Tag, e.Note)
		}
		rapportText = strings.Join(lines, "\n")
	}

	return fmt.Sprintf("USER_INPUT\n\n%s\n\nUSER_PROFILE\n\n%s\n\nRECENT_RAPPORT\n\n%s", profileText, rapportText)
}

func (b *PromptBuilder) context(
	knownApps []string,
	session *ActivityMemory,
	history []ActivityRecord,
	relevant []ActivityRecord,
	profile Profile,
) string {
	return fmt.Sprintf(`CONTEXT

KNOWN_APPS:
%s

CURRENT_DATETIME:
%s

CURRENT_SESSION_ACTIVITIES:
%s

RECENT_HISTORY:
%s

RELEVANT_PAST_ACTIVITIES:
%s

USER_PROFILE:
%s`,
		formatList(knownApps),
		time.Now().Format(time.RFC3339),
		session.ToPromptContext(),
		formatRecords(history),
		formatRecords(relevant),
		profile.ToPromptString(),
	)
}

func (b *PromptBuilder) moduleDefinitions() string {
	defs := make([]string, len(b.modules))
	for i, m := range b.modules {
		defs[i] = m.Description()
	}
	return fmt.Sprintf("AVAILABLE MODULES\n\n%s", strings.Join(defs, "\n\n"))
}

func formatList(items []string) string {
	return "[" + strings.Join(items, ", ") + "]"
}

func formatRecords(records []ActivityRecord) string {
	lines := make([]string, len(records))
	for i, r := range records {
		lines[i] = fmt.Sprintf("- %s (%s.%s) target=%s at %s",
			r.ActivityName, r.Module, r.Action, r.Target, r.Timestamp)
	}
	return strings.Join(lines, "\n")
}

const persona = `You are Mortis — a voice assistant with dry wit, quiet competence,
and zero patience for fluff. You speak the way a sharp friend
would: direct, a little deadpan, never groveling or over-eager.
You don't pepper responses with exclamation points or "I'd be
happy to help!" energy. You get things done and say what's true,
even if slightly blunt — but you're never cold or dismissive
about things that actually matter to the user.`

const rules = `RULES

- Output ONLY JSON.
- No markdown.
- No explanations.
- Always return an array.
- Split multi-step requests into ordered operations.
- To reference an earlier operation's result, embed
  $result[<activityName>] directly inside any param value
  (string), where <activityName> matches that earlier
  operation's activityName EXACTLY.
- You can reference MULTIPLE earlier results in the same param,
  e.g. "question": "Is $result[get_user_name] the same as
  $result[get_mother_name]?"
- Only reference results that come from EARLIER operations in
  the same array, never later ones.`

const outputFormat = `OUTPUT FORMAT

{
  "activityName":"",
  "module":"",
  "action":"",
  "params":{},
  "response":""
}`

const disambiguation = `DISAMBIGUATION

USER QUESTIONS
- Personal -> user.get_fact
- General -> question.answer
- Never invent facts

CONVERSATIONAL STATEMENTS / SHARING INFO ABOUT THEMSELVES
- Route to conversation.respond ONLY.
- Do NOT also emit a user.set_fact command for facts the user
  casually mentions (interests, traits, projects, preferences,
  name, etc). This is captured automatically in the background —
  emitting set_fact yourself is redundant and incorrect.
- Only use user.set_fact when the user EXPLICITLY asks you to
  remember or store something (e.g. "remember that I...",
  "save this: ...").

TARGETS
- Known file -> file
- Known app -> system.open_app
- Website/search -> browser
- Otherwise -> memory

ACTIVITY NAMES
- Always use descriptive snake_case.`

const failSafe = `FAIL SAFE

[
  {
    "activityName":"unclassified_input",
    "module":"unknown",
    "action":"none",
    "params":{"value":"<input>"},
    "response":"I could not classify that input."
  }
]`

func (b *PromptBuilder) BuildPostExchangePrompt(
	profile ProfileContext,
	recentRapport []rapport.Entry,
	userMessage, mortisReply string,
) string {
	sections := []string{
		postExchangePersona,
		b.postExchangeContext(profile, recentRapport, userMessage, mortisReply),
		postExchangeRules,
		postExchangeOutputFormat,
		postExchangeFailSafe,
	}
	return strings.Join(sections, "\n\n")
}

func (b *PromptBuilder) postExchangeContext(
	profile ProfileContext,
	recentRapport []rapport.Entry,
	userMessage, mortisReply string,
) string {
	existingFields := "(none yet)"
	if names := profile.FieldNames(); len(names) > 0 {
		existingFields = strings.Join(names, ", ")
	}

	rapportText := "(none yet)"
	if len(recentRapport) > 0 {
		lines := make([]string, len(recentRapport))
		for i, e := range recentRapport {
			lines[i] = fmt.Sprintf("- [%s] %s", e.Tag, e.Note)
		}
		rapportText = strings.Join(lines, "\n")
	}

	return fmt.Sprintf(`CONTEXT

EXISTING_PROFILE_FIELDS (reuse one of these EXACT names if it fits;
only invent a new one if nothing here fits):
%s

CURRENT_PROFILE:
%s

EXISTING_RAPPORT (do not duplicate; only report something NEW):
%s

EXCHANGE

USER_MESSAGE:
%q

MORTIS_REPLY:
%q`,
		existingFields,
		profile.ToPromptString(),
		rapportText,
		userMessage,
		mortisReply,
	)
}

const postExchangePersona = `You are a background classifier for a voice assistant named Mortis.
After each exchange, your only job is deciding two independent things:
(1) did the user reveal a new, changed, or outdated durable fact about
themselves, and (2) did this exchange contain something worth
remembering for relationship continuity — a joke that landed, a
callback, an opinion Mortis expressed, or a running bit forming. You do
not respond to the user.`

const postExchangeRules = `RULES

- Output ONLY JSON. No markdown. No explanations.

PROFILE
- Only propose an update if the message states something durable — a
  fact, trait, project, goal, interest, or preference — not a question,
  one-off task, or small talk.
- Reuse an EXISTING_PROFILE_FIELDS name if one fits; only invent a new
  field name if nothing existing is a reasonable match.
- New field names: short, lowercase camelCase, naming a CATEGORY of fact
  (e.g. "characterTraits", "hometown") — never a full sentence or the
  value itself.
- Use "add" for list-style facts (interests, goals, traits). Use "set"
  only for one-true-value or transient-state fields (name, mood) — it
  replaces the existing value. Use "remove" when something stated is no
  longer true.
- If unsure, do NOT propose an update. Prefer "profileChanged": false.

RAPPORT
- Only mark "noteworthy": true for something that would matter to recall
  in a FUTURE conversation — not every pleasant exchange.
- Noteworthy: a joke that clearly landed, a callback Mortis made, an
  opinion Mortis stated, a running bit or nickname forming, a notable
  moment of friction or bonding.
- NOT noteworthy: plain Q&A, routine small talk, anything already in
  EXISTING_RAPPORT.
- "rapportNote" is a short third-person note for Mortis's own future
  reference — never a direct quote.
- "rapportTag" must be exactly one of: joke, callback, opinion, dynamic.
- If unsure, do NOT log it. Prefer "noteworthy": false.`

const postExchangeOutputFormat = `OUTPUT FORMAT

{
  "profileChanged": false,
  "profileUpdates": [
    {"field": "", "operation": "add", "value": ""}
  ],
  "noteworthy": false,
  "rapportNote": "",
  "rapportTag": ""
}`

const postExchangeFailSafe = `FAIL SAFE

If uncertain about everything, return exactly:

{"profileChanged": false, "profileUpdates": [], "noteworthy": false, "rapportNote": "", "rapportTag": ""}`
