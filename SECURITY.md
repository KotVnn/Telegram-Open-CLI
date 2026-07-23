# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability within TOC, please send an email to the project maintainer. All security vulnerabilities will be promptly addressed.

**Please do NOT report security vulnerabilities through public GitHub issues.**

### What to include

When reporting a vulnerability, please include:

1. **Description** - A clear description of the vulnerability
2. **Steps to reproduce** - How to reproduce the issue
3. **Impact** - What is the potential impact
4. **Suggested fix** - If you have a suggested fix (optional)

### Response timeline

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 1 week
- **Fix released**: Depends on severity

## Security Measures

### Data Storage

- API keys are stored in the configuration file with restricted permissions
- Database files have restricted file permissions (0600)
- Sensitive data is never logged
- No telemetry is collected without user consent

### Authentication

- Telegram user ID validation for all requests
- Optional admin-only access mode
- Rate limiting on all endpoints
- Session ownership validation

### Input Validation

- All user inputs are sanitized
- Command injection prevention
- File upload size limits (20MB)
- File type validation

### Communication

- HTTPS required for webhook mode
- No data sent to third parties
- Minimal data collection

## Security Best Practices for Users

### Configuration

1. **Restrict bot access** - Configure `allowed_users` in config
2. **Use webhook mode** - More secure than polling in production
3. **Set file permissions** - `chmod 600 ~/.toc/config.toml`
4. **Don't share tokens** - Keep your bot token secret

### Deployment

1. **Use HTTPS** - For webhook mode
2. **Firewall** - Restrict access to necessary ports only
3. **Updates** - Keep TOC updated to latest version
4. **Monitoring** - Monitor logs for suspicious activity

### API Keys

1. **Environment variables** - Prefer over config file
2. **Key rotation** - Rotate API keys regularly
3. **Minimal permissions** - Use least privilege principle
4. **No hardcoded keys** - Never commit keys to source control

## Known Security Considerations

### Telegram Bot API

- Bot tokens provide full access to the bot
- Messages are sent over HTTPS to Telegram servers
- Telegram stores message history according to their policies

### AI Backend Integration

- Prompts and responses are sent to AI providers
- Code context may be included in prompts
- Users should understand their AI provider's data policies

### Local Storage

- SQLite database contains conversation history
- Database file should be protected with file permissions
- Regular backups recommended

## Updates

Security updates will be released as soon as possible after a vulnerability is confirmed. Updates will be announced via:

- GitHub Releases
- CHANGELOG.md
- Security advisories (for critical issues)

## Credits

We thank all security researchers who responsibly disclose vulnerabilities.

## Contact

For security concerns, please contact:
- Email: kotvnn@msn.com
- GitHub Security Advisories: https://github.com/KotVnn/Telegram-Open-CLI/security/advisories
