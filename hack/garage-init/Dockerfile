# Minimal container for the example Python app using `uv` as package manager
ARG PYTHON_IMAGE=python:3.11.14-slim
ARG UV_VERSION=""

# Builder: create a deterministic `.venv` using uv and the lockfile.
FROM ${PYTHON_IMAGE} AS builder
ARG UV_VERSION
WORKDIR /src

# Ensure pip is recent
RUN python -m pip install --upgrade pip setuptools wheel

# Install uv. If you want to pin a uv version, pass --build-arg UV_VERSION=<version>
RUN if [ -n "${UV_VERSION}" ]; then \
			python -m pip install --no-cache-dir "uv==${UV_VERSION}"; \
		else \
			python -m pip install --no-cache-dir uv; \
		fi

# Copy project metadata (we REQUIRE a `uv.lock` to ensure strict, reproducible builds).
# This enforces that you generate a lockfile locally via `uv lock` before building.
COPY pyproject.toml uv.lock ./

# Create the virtual environment and install exact versions from `uv.lock`.
# Note: older flag `--no-interaction` is not supported by uv; run plain `uv sync`
# which is non-interactive when a lockfile (`uv.lock`) is present.
RUN uv sync


# Final image: copy the prepared venv and the application code.
FROM ${PYTHON_IMAGE} AS runtime
WORKDIR /app

# Copy the locked virtual environment from the builder stage
COPY --from=builder /src/.venv .venv

# Copy application files
COPY app.py ./
COPY README.md ./
COPY config.py ./

ENV PYTHONUNBUFFERED=1
ENV PATH=/app/.venv/bin:$PATH

# Run the app with the pinned interpreter and pinned dependencies from `.venv`.
ENTRYPOINT ["/app/.venv/bin/python", "app.py"]
