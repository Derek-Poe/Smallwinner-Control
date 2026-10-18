package dev.derekpoe.smallwinner;

import org.junit.Test;
import static org.junit.Assert.*;

public class TrustedContentTest {
    @Test public void permitsOnlyContractResources() {
        String[] paths = {"/", "/app.js", "/style.css", "/api/v1/state", "/api/v1/config",
                "/api/v1/simulation", "/api/v1/actions/restore"};
        for (String path : paths) assertTrue(path, TrustedContent.allowsResource(TrustedContent.ORIGIN + path));
        assertTrue(TrustedContent.allowsNavigation(TrustedContent.HOME));
        for (String path : paths) if (!path.equals("/")) {
            assertFalse(path, TrustedContent.allowsNavigation(TrustedContent.ORIGIN + path));
        }
    }

    @Test public void rejectsOriginAndParserConfusion() {
        String[] urls = {null, "", "http://127.0.0.1:1717", "http://127.0.0.1:1718/",
                "http://127.0.0.1/", "https://127.0.0.1:1717/", "http://localhost:1717/",
                "http://127.1:1717/", "http://2130706433:1717/", "http://[::1]:1717/",
                "http://127.0.0.1.example.invalid:1717/", "http://127.0.0.1:1717@evil.invalid/",
                "http://evil.invalid@127.0.0.1:1717/", "http://127.0.0.1:01717/",
                "HTTP://127.0.0.1:1717/", "http://127.0.0.1:1717/\\evil.invalid",
                "http://127.0.0.1:1717/../app.js", "http://127.0.0.1:1717/%61pp.js",
                "http://127.0.0.1:1717/api%2fv1/state", "http://127.0.0.1:1717//app.js",
                "http://127.0.0.1:1717/?next=https://evil.invalid/", "http://127.0.0.1:1717/#fragment",
                "http://127.0.0.1:1717/app.js?", "http://127.0.0.1:1717/api/v1/unknown",
                "http://127.0.0.1:1717/\n", "file:///android_asset/offline/index.html",
                "content://example/document", "javascript:alert(1)", "intent://open", "data:text/html,test",
                TrustedContent.OFFLINE, "https://example.invalid/"};
        for (String url : urls) {
            assertFalse(String.valueOf(url), TrustedContent.allowsResource(url));
            assertFalse(String.valueOf(url), TrustedContent.allowsNavigation(url));
        }
    }
}
