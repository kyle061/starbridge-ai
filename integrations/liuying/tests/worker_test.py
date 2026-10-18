import importlib.util
import ipaddress
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("resolver", Path(__file__).parents[1] / "worker/resolve.py")
resolver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(resolver)


class NetworkGuardTests(unittest.TestCase):
    def test_normalizes_douyin_share_and_modal_urls(self):
        self.assertEqual(resolver.canonical_douyin("https://www.iesdouyin.com/share/video/123456/"), "https://www.douyin.com/video/123456")
        self.assertEqual(resolver.canonical_douyin("https://www.douyin.com/?modal_id=123456"), "https://www.douyin.com/video/123456")
        with self.assertRaises(ValueError):
            resolver.canonical_douyin("https://douyin.com.evil.test/video/123456")

    def test_identifies_douyin_image_posts(self):
        with self.assertRaises(ValueError) as error:
            resolver.canonical_douyin("https://www.douyin.com/note/123456")
        self.assertEqual(resolver.classify_error(str(error.exception))[0], "NOT_VIDEO")

    def test_blocks_private_loopback_linklocal_and_mapped_ipv4(self):
        for address in ["127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "::1", "fe80::1", "::ffff:127.0.0.1"]:
            with self.subTest(address=address), patch.object(resolver, "_getaddrinfo", return_value=[(2, 1, 6, "", (address, 443))]):
                with self.assertRaises(OSError):
                    resolver.public_addresses("example.com", 443)

    def test_rejects_mixed_dns_answers(self):
        with patch.object(resolver, "_getaddrinfo", return_value=[(2, 1, 6, "", ("8.8.8.8", 443)), (2, 1, 6, "", ("10.1.1.1", 443))]):
            with self.assertRaises(OSError):
                resolver.public_addresses("example.com", 443)

    def test_accepts_public_resolution_without_a_second_lookup(self):
        result = [(2, 1, 6, "", ("8.8.8.8", 443))]
        with patch.object(resolver, "_getaddrinfo", return_value=result) as lookup:
            self.assertEqual(resolver.public_addresses("example.com", 443), result)
            lookup.assert_called_once()


if __name__ == "__main__":
    unittest.main()
