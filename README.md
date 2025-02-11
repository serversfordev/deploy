# deploy

A lightweight, single-binary deployment tool written in Go.

Key features:
- Single statically linked binary with zero dependencies
- Simple configuration with minimal boilerplate
- Pull-based deployment strategy
- Easy integration with cron jobs, webhooks, and git hooks
- Atomic deployments with rollback capability

### Why?

`deploy` was born from a desire (and from my personal frustrations) to have a simple deployment tool that:
- works without external dependencies
- is easy to integrate with existing workflows
- is easy to configure
- doesn't require complex server setup
- perfect for personal/small projects

## Table of Contents

- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Hooks](#hooks)
- [Deployment lifecycle](#deployment-lifecycle)

## Quick start

### Installation

```bash
# Download the latest linux-amd64 binary
curl -L https://github.com/serversfordev/deploy/releases/latest/download/deploy-linux-amd64 -o deploy

# Make it executable
chmod +x deploy

# Move it to a directory in your PATH
sudo mv deploy /usr/local/bin/deploy

# Verify installation
deploy --help
```

For other architectures, please see the [releases page](https://github.com/serversfordev/deploy/releases).

### Initializing deployment directory structure

Run the `deploy init` command as the user that your application runs as.

```bash
# Run as the user that your application runs as
sudo -u www-data bash
cd /var/www

# Initialize the deployment directory structure
deploy init --name app-name
```

The command will create the following directory structure:

```
/var/www/app-name/
├── config.toml
├── current -> ./releases/20240209123456
├── releases/
│   ├── 20240209123456/
│   ├── 20240209123400/
│   └── 20240209123000/
├── shared/
│   ├── .env
│   └── storage/
└── logs/
```

### Configuration

```toml
[source]
  provider = "git"
  [source.git]
    repo = "https://github.com/yourname/app-name.git" # Set the git repository
    branch = "main" # And the branch

[deploy]
  keep_releases = 3
  [deploy.jitter]
    min = 5
    max = 10
  [deploy.shared]
    dirs = []
    files = []
```

### Deploying

Enter the deployment directory and run the `deploy start` command.

```bash
# Enter the deployment directory
cd /var/www/app-name

# Start the deployment
deploy start
```

You can trigger the start command from a cron job, a [webhook](https://github.com/adnanh/webhook), or a git hook.

## Configuration

The configuration file (`config.toml`) defines how your application should be deployed. Here's a detailed explanation of each option:

### Source configuration

```toml
[source]
  provider = "git"
  [source.git]
    repo = "https://github.com/yourname/app-name.git"
    branch = "main"
```

- `provider`: The source provider for your application (currently only "git" is supported)
- `repo`: The Git repository URL of your application
- `branch`: The branch to deploy from (defaults to "main")

### Deployment settings

```toml
[deploy]
  keep_releases = 3
  [deploy.jitter]
    min = 5
    max = 10
  [deploy.shared]
    dirs = []
    files = []
```

#### General settings

keep_releases: Number of releases to keep in the releases directory (defaults to 3)

#### Jitter settings

The jitter settings add a random delay before deployment to prevent multiple servers from deploying simultaneously:

- `min`: Minimum delay in seconds before deployment starts
- `max`: Maximum delay in seconds before deployment starts

#### Shared resources

The shared resources section defines files and directories that should be copied to the new release:

Configure files and directories that should be shared between releases:

- `dirs`: List of directories to be shared (e.g., ` ["storage", "uploads"]`)
- `files`: List of files to be shared (e.g., `[".env"]`)

## Hooks

Hooks allow you to customize the deployment process. Place your hook scripts in the .deploy/hooks directory in your application's repository. All hooks must be executable.

```
your-app/
└── .deploy/
    └── hooks/
        ├── clone
        ├── build
        ├── deploy
        ├── post_deploy
        └── verify
```

### Available hooks

- `clone`: Runs after the code is cloned into a new release directory, but before the shared resources are linked
- `build`: Main build process (compile assets, install dependencies)
- `deploy`: Runs during the deployment phase (before the current symlink is updated)
- `post_deploy`: Runs after deployment is complete
- `verify`: Runs verification checks after deployment

### Hook example
Here's an example `build` hook for a Laravel application:

```
#!/bin/bash

set -e

echo "Building"

mkdir -p storage/{app/public,framework/{cache,sessions,testing,views},logs}

composer install --no-dev

php artisan optimize:clear
php artisan optimize

php artisan storage:link

php artisan migrate --force

npm install
npm run build
```

The hooks should be placed in your application's `.deploy/hooks` directory and must be executable (`chmod +x .deploy/hooks/build`).

## Deployment lifecycle

The deployment process follows a specific lifecycle with multiple stages, described as a state machine under the `internal/deployer/deployer.go` file.

### 1. Initialize

- Acquires deployment lock
- Applies jitter delay if configured
- Initializes the source provider


### 2. Detect Changes

- Compares current release with remote source
- Proceeds if changes detected or force flag is set
- Skips deployment if no changes and no force flag


### 3. Clone

- Creates new release directory
- Clones the repository
- Executes the `clone` hook
- Links shared files and directories


### 4. Build

- Executes the `build` hook
- Typically used for compiling assets, installing dependencies
- Must exit with 0 for deployment to continue


### 5. Deploy

- Executes the `deploy` hook
- Updates the current symlink to point to the new release
- Prepares rollback in case of subsequent failures


### 6. Post-Deploy

- Executes the `post_deploy` hook
- Used for tasks that should run after the deployment is live
- Example: cache warming, service notifications


### 7. Verify

- Executes the `verify` hook
- Validates the deployment
- Non-zero exit triggers automatic rollback


### 8. Error

- Rollback is executed on error in any state
- Reverts to the previous release


### 9. Finalize

- Cleans up old releases
- Releases deployment lock
