import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';

void main() {
  test('prefers Wi-Fi/Ethernet over cellular without assuming 192.168/16', () {
    final selected = ServiceManager.selectUsableLanIpv4(const [
      LanAddressCandidate(interfaceName: 'rmnet0', address: '172.20.4.9'),
      LanAddressCandidate(interfaceName: 'wlan0', address: '10.0.0.24'),
    ]);

    expect(selected, '10.0.0.24');
  });

  test('rejects bind, loopback, link-local, multicast, and IPv6 addresses', () {
    final selected = ServiceManager.selectUsableLanIpv4(const [
      LanAddressCandidate(interfaceName: 'wlan0', address: '0.0.0.0'),
      LanAddressCandidate(interfaceName: 'lo', address: '127.0.0.1'),
      LanAddressCandidate(interfaceName: 'wlan0', address: '169.254.8.2'),
      LanAddressCandidate(interfaceName: 'wlan0', address: '224.0.0.1'),
      LanAddressCandidate(interfaceName: 'wlan0', address: '::1'),
    ]);

    expect(selected, isNull);
  });

  test(
    'prefers private LAN address over a public address on same interface',
    () {
      final selected = ServiceManager.selectUsableLanIpv4(const [
        LanAddressCandidate(interfaceName: 'en0', address: '203.0.113.8'),
        LanAddressCandidate(interfaceName: 'en0', address: '192.168.50.7'),
      ]);

      expect(selected, '192.168.50.7');
    },
  );
}
