# blog/BUILD

files(
    name="src_files",
    sources=[
        "src/**/*",
        "public/**/*",
        "astro.config.mjs",
        "package.json",
        "package-lock.json",
        "tsconfig.json",
        "!node_modules/**",
        "!dist/**",
        "!.astro/**",
    ],
)

experimental_test_shell_command(
    name="test",
    command="npm ci && npm run build",
    tools=["npm", "node", "npx", "bash", "sh", "env"],
    execution_dependencies=[":src_files"],
    workdir="blog",
)

shell_command(
    name="dist",
    command="npm ci && npm run build",
    tools=["npm", "node", "npx", "bash", "sh", "env"],
    execution_dependencies=[":src_files"],
    output_directories=["dist"],
    workdir="blog",
)
