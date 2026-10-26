schema_version = 1

project {
  license          = "BUSL-1.1"
  copyright_holder = "Dit"
  copyright_year   = 2026

  header_ignore = [
    "gradlew",
    "gradlew.bat",
    "gradle/**",
    "build/**",
    "**/build/**",
    ".health/**",
    "**/*.out",
    # Only Dit-authored files carry the Dit header. Anything vendored from
    # another project lives under a third_party/ directory with its own
    # LICENSE and is never header-checked.
    "**/third_party/**",
  ]
}
