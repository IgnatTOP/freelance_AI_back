# PowerShell скрипт для сброса базы данных
# Использование: .\reset_db.ps1

# Загружаем переменные окружения из .env
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            $name = $matches[1].Trim()
            $value = $matches[2].Trim()
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
        }
    }
}

# Получаем DATABASE_URL из переменных окружения
$databaseUrl = $env:DATABASE_URL
if (-not $databaseUrl) {
    $databaseUrl = "postgres://postgres:123@localhost:5432/freelance_ai?sslmode=disable"
    Write-Host "DATABASE_URL не найден, используем значение по умолчанию" -ForegroundColor Yellow
}

# Парсим DATABASE_URL
if ($databaseUrl -match 'postgres://([^:]+):([^@]+)@([^:]+):(\d+)/([^?]+)') {
    $username = $matches[1]
    $password = $matches[2]
    $dbHost = $matches[3]
    $port = $matches[4]
    $database = $matches[5]
    
    Write-Host "Подключение к базе данных: ${dbHost}:${port}/${database}" -ForegroundColor Cyan
    
    # Проверяем наличие psql
    $psqlPath = Get-Command psql -ErrorAction SilentlyContinue
    if (-not $psqlPath) {
        # Пытаемся найти psql в стандартных местах установки PostgreSQL на Windows
        $possiblePaths = @(
            "C:\Program Files\PostgreSQL\*\bin\psql.exe",
            "C:\Program Files (x86)\PostgreSQL\*\bin\psql.exe",
            "$env:LOCALAPPDATA\Programs\PostgreSQL\*\bin\psql.exe"
        )
        
        $found = $false
        foreach ($pathPattern in $possiblePaths) {
            $foundPath = Get-ChildItem -Path $pathPattern -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($foundPath) {
                $psqlPath = $foundPath.FullName
                $found = $true
                Write-Host "Найден psql: $psqlPath" -ForegroundColor Green
                break
            }
        }
        
        if (-not $found) {
            Write-Host "ОШИБКА: psql не найден." -ForegroundColor Red
            Write-Host "Попробуйте один из следующих вариантов:" -ForegroundColor Yellow
            Write-Host "1. Добавьте PostgreSQL в PATH" -ForegroundColor Yellow
            Write-Host "2. Установите PostgreSQL: https://www.postgresql.org/download/windows/" -ForegroundColor Yellow
            Write-Host "3. Используйте альтернативный способ сброса базы данных" -ForegroundColor Yellow
            exit 1
        }
    } else {
        $psqlPath = $psqlPath.Source
    }
    
    # Устанавливаем переменную окружения для пароля
    $env:PGPASSWORD = $password
    
    # Выполняем SQL скрипт
    Write-Host "Выполняю сброс базы данных..." -ForegroundColor Yellow
    $sqlScript = Join-Path $PSScriptRoot "reset_db.sql"
    
    $result = & $psqlPath -h $dbHost -p $port -U $username -d $database -f $sqlScript 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "База данных успешно сброшена!" -ForegroundColor Green
        Write-Host "Запустите сервер, чтобы применить миграции заново." -ForegroundColor Cyan
    } else {
        Write-Host "ОШИБКА при сбросе базы данных:" -ForegroundColor Red
        Write-Host $result
        exit 1
    }
    
    # Очищаем пароль из переменных окружения
    Remove-Item Env:\PGPASSWORD
} else {
    Write-Host "ОШИБКА: Неверный формат DATABASE_URL" -ForegroundColor Red
    Write-Host "Ожидается формат: postgres://user:password@host:port/database" -ForegroundColor Yellow
    exit 1
}

