import PostHog from 'posthog-react-native';

class AnalyticsService {
  private posthog: PostHog | null = null;

  async init(): Promise<void> {
    const apiKey = process.env.EXPO_PUBLIC_POSTHOG_API_KEY;
    if (!apiKey) {
      console.log('[POSTHOG INFO] EXPO_PUBLIC_POSTHOG_API_KEY not set, using console fallback');
      return;
    }
    try {
      this.posthog = new PostHog(apiKey, {
        host: process.env.EXPO_PUBLIC_POSTHOG_HOST || 'https://us.i.posthog.com',
        flushInterval: 10,
        flushAt: 1,
      });
      console.log('[POSTHOG SUCCESS] PostHog Analytics initialized successfully');
    } catch (err) {
      console.log('[POSTHOG WARNING] PostHog running in local fallback mode:', err);
    }
  }

  // Track user actions, screen views, and custom driver events
  track(eventName: string, properties?: Record<string, any>): void {
    if (this.posthog) {
      this.posthog.capture(eventName, properties);
    }
    if (typeof __DEV__ !== 'undefined' && __DEV__) {
      console.log(`[ANALYTICS EVENT] ${eventName}:`, JSON.stringify(properties || {}));
    }
  }

  // Identify driver user session
  identify(driverId: string, traits?: Record<string, any>): void {
    if (this.posthog) {
      this.posthog.identify(driverId, traits);
    }
    if (typeof __DEV__ !== 'undefined' && __DEV__) {
      console.log(`[ANALYTICS IDENTIFY] Driver: ${driverId}`, JSON.stringify(traits || {}));
    }
  }
}

export const Analytics = new AnalyticsService();
