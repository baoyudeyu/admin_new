@echo off
echo ==========================================
echo          Deploy to GitHub Repository  
echo ==========================================
echo.

set REPO_URL=https://github.com/baoyudeyu/admin_new

if not exist ".git" (
    echo [INFO] Initializing Git repository...
    git init
    if errorlevel 1 (
        echo [ERROR] Git initialization failed
        pause
        exit /b 1
    )
)

git remote get-url origin >nul 2>&1
if errorlevel 1 (
    echo [INFO] Adding remote repository...
    git remote add origin %REPO_URL%
) else (
    echo [INFO] Updating remote repository URL...
    git remote set-url origin %REPO_URL%
)

for /f "tokens=*" %%i in ('git branch --show-current 2^>nul') do set CURRENT_BRANCH=%%i
if "%CURRENT_BRANCH%"=="" (
    echo [INFO] Creating main branch...
    git checkout -b main
)

echo [INFO] Adding files to staging area...
git add .

git diff --cached --quiet
if not errorlevel 1 (
    echo [WARNING] No changes detected
    echo [INFO] Force pushing current state...
) else (
    echo [INFO] Committing changes...
    git commit -m "Auto deploy: %date% %time%"
)

echo [INFO] Pushing to GitHub repository...
git push -u origin main --force

if errorlevel 1 (
    echo.
    echo [ERROR] Push failed. Possible reasons:
    echo 1. Network connection issues
    echo 2. Repository permission issues  
    echo 3. Git authentication issues
    echo.
    echo Please check:
    echo - GitHub repository exists and has write permission
    echo - Git username and email are configured
    echo - SSH key or personal access token configured
    echo.
    pause
    exit /b 1
)

echo.
echo ==========================================
echo          Deploy Success!
echo ==========================================
echo Repository: %REPO_URL%
echo Time: %date% %time%
echo ==========================================
echo.

pause 