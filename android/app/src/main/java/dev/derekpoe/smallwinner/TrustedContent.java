package dev.derekpoe.smallwinner;

import java.net.URI;
import java.net.URISyntaxException;
import java.util.Set;

/** Literal URL contract shared with the default simulator. No configurable remote origin. */
public final class TrustedContent {
    public static final String ORIGIN = "http://127.0.0.1:1717";
    public static final String HOME = ORIGIN + "/";
    public static final String OFFLINE = "https://offline.smallwinner.invalid/";
    private static final Set<String> PATHS = Set.of("/", "/app.js", "/style.css",
            "/api/v1/state", "/api/v1/config", "/api/v1/simulation", "/api/v1/actions/restore");

    private TrustedContent() { }

    public static boolean allowsResource(String value) {
        if (value == null || value.length() > 256) return false;
        try {
            URI uri = new URI(value);
            return "http".equals(uri.getScheme()) && "127.0.0.1".equals(uri.getHost())
                    && uri.getPort() == 1717 && uri.getRawUserInfo() == null
                    && uri.getRawQuery() == null && uri.getRawFragment() == null
                    && PATHS.contains(uri.getRawPath())
                    && value.equals(ORIGIN + uri.getRawPath());
        } catch (URISyntaxException ignored) {
            return false;
        }
    }

    public static boolean allowsNavigation(String value) { return HOME.equals(value); }
}
