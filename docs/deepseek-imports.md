# DeepSeek Vision для расписаний и чеков

Общий серверный клиент использует `deepseek-flash`, `thinking: {"type":"disabled"}` и JSON mode. Фото передаётся в теле запроса как base64, без публичной ссылки и без сохранения в Files API провайдера. Данные передаются DeepSeek только после явного подтверждения пользователя в форме импорта. Результат AI недоверенный: каждый импорт проверяет свою схему, даты и суммы и требует просмотра перед сохранением.

Настройки сервера:

- `SRUNAL_DEEPSEEK_API_KEY`: секрет из окружения или хранилища секретов, не из исходного кода.
- `SRUNAL_DEEPSEEK_BASE_URL`: по умолчанию `https://api.deepseek.com`.
- `SRUNAL_DEEPSEEK_MODEL`: по умолчанию `deepseek-flash`.

Режим размышления всегда отключён в клиенте. Ключ никогда не отправляется браузеру, не должен иметь префикс `VITE_` и не должен попадать в Docker image или сборку frontend. Compose передаёт его только процессу API через окружение. Отсутствие ключа честно отключает распознавание; ручной ввод остаётся доступным.

Локально на Windows ключ можно хранить за пределами проекта в `%LOCALAPPDATA%\SruNal\secrets\deepseek-api.dpapi`. Это строка, зашифрованная Windows DPAPI для текущего пользователя; файл и папка доступны этому пользователю. Перед запуском API загрузить значение только в окружение текущего процесса:

```powershell
$secretPath = Join-Path $env:LOCALAPPDATA 'SruNal\secrets\deepseek-api.dpapi'
$secure = ConvertTo-SecureString (Get-Content -LiteralPath $secretPath -Raw)
$env:SRUNAL_DEEPSEEK_API_KEY = [System.Net.NetworkCredential]::new('', $secure).Password
# Далее обычный запуск сервера или docker compose в этом же процессе.
```

Не печатать переменную, HTTP Authorization, содержимое `docker compose config` с подставленными секретами или приватное фото в логи. Проверять Compose можно через `docker compose config --quiet`. В production использовать отдельное хранилище секретов, а не Windows DPAPI-файл с чужой машины.

Клиент ограничивает размер изображения 10 МиБ, число параллельных запросов и время ожидания. Он не повторяет платные запросы автоматически, не следует redirect и не возвращает клиенту тело ошибки провайдера, которое может содержать приватные данные. Ответы с пустым/оборванным JSON отвергаются.

Официальные контракты: [Vision](https://api-docs.deepseek.com/guides/vision.md), [Thinking mode](https://api-docs.deepseek.com/guides/thinking_mode.md), [JSON output](https://api-docs.deepseek.com/guides/json_mode.md).
