# apps/metrics-worker/BUILD

python_sources(
    name="metrics-worker",
)

files(
    name="files",
    sources=[
        "worker.py",
        "pyproject.toml",
        "wrangler.toml",
        "!BUILD",
    ],
)

archive(
    name="dist",
    format="tar.gz",
    files=[":files"],
    output_path="apps/metrics-worker.tar.gz",
)
