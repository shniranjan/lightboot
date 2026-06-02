# Troubleshooting

This page covers common issues you might encounter when setting up and running LightBoot, along with their solutions.

---

## Quick Diagnostic Checklist

Before diving into specific issues, run through this checklist:

1. &#x2705; Is LightBoot running? `curl http://localhost:8080/api/health`
2. &#x2705; Is the API token valid? Test with `curl -H "Authorization: Bearer TOKEN" http://localhost:8080/api/isos`
3. &#x2705; Are the ISOs detected? Check the Web UI ISO Library or `/api/isos`
4. &#x2705; Are the ISOs in `ready` status? If `processing` or `error`, check logs
5. &#x2705; Can clients reach the server? Check firewall rules for ports 67, 69, 8080
6. &#x2705; Is the DHCP proxy running? `sudo ss -ulpn | grep 67`

---

## Startup Issues

### "Address already in use" on port 8080

```
Error: listen tcp :8080: bind: address already in use
```

**Cause**: Another service is using port 8080.

**Solution**:
```bash
# Find what's using the port
sudo ss -tlnp | grep 8080
# Change LightBoot's HTTP port
# In config.yaml:
http_port: 9090
# Or via environment:
export LIGHTBOOT_HTTP_PORT=9090
./lightboot
```

### "Permission denied" on port 67

```
Error: listen udp :67: bind: permission denied
```

**Cause**: Port 67 requires `CAP_NET_BIND_SERVICE` or root.

**Solution**:
```bash
# Grant the capability
sudo setcap cap_net_bind_service=+ep ./lightboot

# Or disable DHCP proxy (if not needed)
# In config.yaml:
dhcp_proxy_enabled: false
```

### "Failed to load profiles"

```
Error loading profiles: yaml: unmarshal errors: ...
```

**Cause**: A custom profile in `profiles/` has invalid YAML syntax.

**Solution**:
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('profiles/my-profile.yml'))"

# Check for common issues:
# - Tabs instead of spaces (YAML requires spaces)
# - Missing quotes around regex patterns
# - Incorrect indentation (must be 2 or 4 spaces, consistent)
```

---

## ISO Detection Issues

### ISO stays "unknown" status

**Possible causes**:
1. No profile matches the distribution
2. ISO filesystem can't be read
3. Detection rules are too strict

**Debugging steps**:
```bash
# 1. Check recent logs
curl -s http://localhost:8080/api/logs/recent \
  -H "Authorization: Bearer TOKEN" | jq '.[] | select(.source=="detector")'

# 2. Verify ISO is readable
xorriso -indev ./iso/my-image.iso -ls 2>&1 | head -20

# 3. List ISO contents
python3 -c "
import isoparser
iso = isoparser.parse('./iso/my-image.iso')
for f in iso.root:
    print(f.name)
"

# 4. Check which files the profile expects
cat profiles/*.yml | grep -A5 "files:"
```

**Solution**: [Write a custom profile](profiles.md#custom-profiles) or check if a profile already exists for your distribution.

### ISO stays "processing" forever

**Cause**: Extraction is stuck (large file, disk I/O, or reader crash).

**Solution**:
```bash
# 1. Check if cache has partial files
ls -la cache/

# 2. Check disk space
df -h ./cache

# 3. Delete the ISO and re-upload
curl -X DELETE http://localhost:8080/api/isos/ID \
  -H "Authorization: Bearer TOKEN"
```

### ISO shows "error" status

**Cause**: Extraction failed.

**Solution**:
```bash
# Check logs for the specific error
curl -s http://localhost:8080/api/logs/recent \
  -H "Authorization: Bearer TOKEN" | jq '.[] | select(.level=="error")'

# Common errors:
# - "disk full" → free up space or increase cache size
# - "file not found in ISO" → profile references wrong paths
# - "read error" → ISO may be corrupt, verify SHA256

# Verify ISO integrity
sha256sum ./iso/my-image.iso
# Compare with official checksums
```

---

## PXE Boot Issues

### Client doesn't receive DHCPOFFER from LightBoot

**Debugging**:
```bash
# On the LightBoot server, capture DHCP traffic
sudo tcpdump -i any port 67 or port 68 -v -n

# Look for:
# - DHCPDISCOVER from client MAC address
# - DHCPOFFER from LightBoot's IP
# If you see DISCOVER but no OFFER, the proxy isn't responding
```

**Solutions**:
- Verify `dhcp_proxy_enabled: true` in config
- Check that the server and client are on the same subnet/VLAN
- Ensure no firewall is blocking port 67
- Try increasing log level to `debug` for DHCP diagnostics

### Client gets DHCPOFFER but no TFTP transfer

**Debugging**:
```bash
# Test TFTP from another machine
echo -ne "\x00\x01undionly.kpxe\x00octet\x00" | timeout 3 nc -u SERVER_IP 69 | wc -c
# If response is 0 bytes, TFTP isn't responding
```

**Solutions**:
- Verify LightBoot is listening: `sudo ss -ulpn | grep 69`
- Check firewall allows UDP 69
- Verify `undionly.kpxe` exists in `bootfiles/`
- Check TFTP logs: `curl /api/logs/recent -H "Authorization: Bearer TOKEN" | grep tftp`

### TFTP works but iPXE doesn't load the menu

**Debugging**:
```bash
# Test the boot script endpoint from the client network
curl http://SERVER_IP:8080/api/boot/ipxe
# Should return #!ipxe script
```

**Solutions**:
- Verify firewall allows TCP 8080 from client subnet
- Check `http_address` config (should not be `127.0.0.1` if clients are remote)
- iPXE uses the DHCP-provided server IP — verify it matches LightBoot's IP

### Kernel boots but panics or hangs

**Possible causes**:
- Wrong architecture (e.g., amd64 ISO booting on arm64 client)
- Missing kernel modules in initrd
- Incorrect kernel command line

**Solutions**:
- Verify the ISO architecture matches the client
- Check the boot menu syntax on the Web UI **Boot Menu** page
- Try adding `debug` or `loglevel=7` to the kernel command line in the profile
- Test the same ISO booted directly (USB/DVD) to verify it's working

---

## Web UI Issues

### Can't log in (invalid token)

```bash
# Verify your token
cat data/.api_token

# Test with curl
curl http://localhost:8080/api/health \
  -H "Authorization: Bearer $(cat data/.api_token)"
# Should return {"status":"ok"}

# If token is lost, regenerate
rm data/.api_token
./lightboot
# New token printed to console
```

### White screen / blank page

**Cause**: The Vue.js SPA hasn't been built, or the embedded files are missing.

**Solution**:
```bash
# Build the frontend
make web
# Rebuild the binary
make build
# Restart
./bin/lightboot
```

### Upload progress bar not working

**Cause**: The progress bar uses `XMLHttpRequest.upload.onprogress`. If the upload is very fast (local network, small ISO), the progress bar might jump from 0% to 100% instantly.

**Solution**: This is expected for small files. For large files, it should work correctly. Try uploading a large (>1 GB) ISO to verify.

### Logs not appearing / SSE connection fails

**Debugging**:
```bash
# Test SSE stream with curl
curl -N http://localhost:8080/api/logs/stream
# Should stream log events

# Test recent logs endpoint
curl http://localhost:8080/api/logs/recent \
  -H "Authorization: Bearer TOKEN"
```

**Solutions**:
- Check browser console for EventSource errors
- SSE is a public endpoint (no auth header) — should work from any client
- Some corporate proxies block SSE; try from a different network

---

## Performance Issues

### Slow ISO scanning

**Cause**: Large ISOs (8+ GB) take time to SHA256 hash.

**Solutions**:
- Increase `scanner_interval` to reduce scan frequency
- Use SSDs for the `iso_dir` and `cache_dir`
- The SHA256 is calculated once and cached in the database

### High memory usage

**Cause**: Large ISO filesystem reads or many concurrent requests.

**Solutions**:
- LightBoot typically uses <100 MB RAM
- If memory grows, check for goroutine leaks by monitoring over time
- Restart periodically if needed (systemd service handles this)

### Disk space running out

```bash
# Check sizes
du -sh iso/ cache/ data/

# Clear the cache (safe, will be rebuilt on next boot)
rm -rf cache/*

# Delete old ISOs from the Web UI or API
```

---

## Database Issues

### Database locked

```
Error: database is locked
```

**Cause**: SQLite allows one writer at a time. Concurrent writes (e.g., scanner + upload) can cause contention.

**Solution**: LightBoot uses WAL mode and handles retries internally. If you see this repeatedly:
```bash
# Check database health
sqlite3 data/lightboot.db "PRAGMA integrity_check;"

# Backup and recreate if corrupt
cp data/lightboot.db data/lightboot.db.backup
rm data/lightboot.db
# Restart LightBoot (database will be recreated)
```

### Database schema migration fails

**Cause**: Manual database modification or version mismatch.

**Solution**:
```bash
# Backup, delete, and let LightBoot recreate
cp data/lightboot.db data/lightboot.db.$(date +%s).bak
rm data/lightboot.db
sudo systemctl restart lightboot
```

---

## Getting Help

If you can't resolve an issue:

1. **Check logs**: `curl http://localhost:8080/api/logs/recent -H "Authorization: Bearer TOKEN"`
2. **Run with debug logging**: Set `log_level: debug` in config
3. **Check GitHub Issues**: [LightBoot Issues](https://github.com/shniranjan/lightboot/issues)
4. **Include in your bug report**:
   - LightBoot version
   - OS and architecture
   - Relevant log entries (with `log_level: debug`)
   - Steps to reproduce
   - Client hardware/VM details
