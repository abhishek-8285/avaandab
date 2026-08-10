import PostHog from 'posthog-react-native';

class AnalyticsService {
  private posthog: PostHog | null = null;

  async init(): Promise<void> {
    try {
      this.posthog = new PostHog('phc_avandab_demo_api_key_placeholder', {
        host: 'https://us.i.posthog.com',
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
    console.log(`[ANALYTICS EVENT] ${eventName}:`, JSON.stringify(properties || {}));
  }

  // Identify driver user session
  identify(driverId: string, traits?: Record<string, any>): void {
    if (this.posthog) {
      this.posthog.identify(driverId, traits);
    }
    console.log(`[ANALYTICS IDENTIFY] Driver: ${driverId}`, JSON.stringify(traits || {}));
  }
}

export const Analytics = new AnalyticsService();
