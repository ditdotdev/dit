package commands

// Subcommand name and shared flag/key string constants. Centralized to
// silence goconst across the package: test files reference these names
// many times each, which inflates the package-wide string count and
// would otherwise force every cobra Use: field, switch case, and flag
// name to be marked as a duplicate literal even though it appears once
// per file in production code.
const (
	subcmdAuth      = "auth"
	subcmdLogin     = "login"
	subcmdLogout    = "logout"
	subcmdStatus    = "status"
	subcmdContext   = "context"
	subcmdInstall   = "install"
	subcmdUninstall = "uninstall"
	subcmdOrg       = "org"
	subcmdRepo      = "repo"
	subcmdRemote    = "remote"
	subcmdList      = "list"

	// Shared flag identifiers used by multiple repo/org subcommands.
	flagServer     = "server"
	flagPrivate    = "private"
	flagPublic     = "public"
	flagPermission = "permission"
	flagRole       = "role"

	// Used both as a cobra flag identifier (clone/context/fork/run --name)
	// and as a JSON key in the org request/response body. Same string,
	// same purpose (the entity's name).
	nameKey = "name"
)
