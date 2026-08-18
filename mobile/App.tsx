import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, ScrollView, TouchableOpacity, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query';
import { Colors, Font, Radius, Spacing } from './src/constants/theme';
import { DEFAULT_DRIVER_ID, DEFAULT_LATITUDE, DEFAULT_LONGITUDE } from './src/constants/network';
import { TripCard, SkeletonLoader } from './src/components/TripCard';
import { LiveDriverTrackingMap } from './src/components/LiveDriverTrackingMap';
import { SplashScreen } from './src/components/SplashScreen';
import { GetStartedScreen } from './src/components/GetStartedScreen';
import { OnboardingOverviewScreen } from './src/components/OnboardingOverviewScreen';
import { BookingScheduleScreen } from './src/components/BookingScheduleScreen';
import { EarningsOverviewScreen } from './src/components/EarningsOverviewScreen';
import { LoginScreen } from './src/components/LoginScreen';
import { RegisterScreen } from './src/components/RegisterScreen';
import { ForgotPasswordScreen } from './src/components/ForgotPasswordScreen';
import { FirstTimeSetupScreen } from './src/components/FirstTimeSetupScreen';
import { DeliveryVerificationScreen } from './src/components/DeliveryVerificationScreen';
import { ActiveNavigationScreen } from './src/components/ActiveNavigationScreen';
import { DB } from './src/services/storage';
import { Telemetry } from './src/services/telemetry';
import { Analytics } from './src/services/analytics';
import { GraphQL } from './src/services/graphql';
import { MQTT } from './src/services/mqtt';
import { SyncEngine } from './src/services/syncEngine';
import { useAuthStore } from './src/stores/authStore';
import { Trip } from './src/types/api';
import { CameraView } from 'expo-camera';

const queryClient = new QueryClient();

// Configurable demo fallbacks; override via EXPO_PUBLIC_* env vars.
const DEMO_DRIVER_ID = DEFAULT_DRIVER_ID || 'DRV-9042';
const DEMO_LATITUDE = DEFAULT_LATITUDE || 18.5204;
const DEMO_LONGITUDE = DEFAULT_LONGITUDE || 73.8567;

export default function App() {
  const { isAuthenticated, isLoading, loadSession } = useAuthStore();
  const [setupCompleted, setSetupCompleted] = useState(false);
  const [currentScreen, setCurrentScreen] = useState<'splash' | 'get_started' | 'onboarding_overview' | 'booking_schedule' | 'earnings_overview' | 'login' | 'register' | 'forgot_password' | 'first_time_setup' | 'active_nav' | 'delivery_verify'>('splash');

  useEffect(() => {
    loadSession();
  }, []);

  if (isLoading) {
    return <SplashScreen onFinish={() => {}} />;
  }

  // Authenticated State -> Load Main App with Navigation Stack & Setup View Access
  if (isAuthenticated) {
    if (!setupCompleted && currentScreen === 'first_time_setup') {
      return (
        <FirstTimeSetupScreen
          onCompleteSetup={() => {
            setSetupCompleted(true);
            setCurrentScreen('active_nav');
          }}
          onBack={() => setSetupCompleted(true)}
        />
      );
    }

    if (currentScreen === 'active_nav') {
      return (
        <ActiveNavigationScreen
          onArriveAtStop={() => setCurrentScreen('delivery_verify')}
          onMenuToggle={() => setSetupCompleted(false)}
        />
      );
    }

    if (currentScreen === 'delivery_verify') {
      return (
        <DeliveryVerificationScreen
          onComplete={() => setCurrentScreen('active_nav')}
          onBack={() => setCurrentScreen('active_nav')}
        />
      );
    }

    return (
      <SafeAreaProvider>
        <QueryClientProvider client={queryClient}>
          <StatusBar style="light" />
          <MainScreen onOpenSetup={() => setCurrentScreen('first_time_setup')} />
        </QueryClientProvider>
      </SafeAreaProvider>
    );
  }

  // Unauthenticated Flow -> Splash / Onboarding / Login / Register
  if (currentScreen === 'splash') {
    return <SplashScreen onFinish={() => setCurrentScreen('get_started')} />;
  }

  if (currentScreen === 'get_started') {
    return (
      <GetStartedScreen
        onGetStarted={() => setCurrentScreen('onboarding_overview')}
        onSignIn={() => setCurrentScreen('login')}
      />
    );
  }

  if (currentScreen === 'onboarding_overview') {
    return (
      <OnboardingOverviewScreen
        onNext={() => setCurrentScreen('booking_schedule')}
        onSkip={() => setCurrentScreen('login')}
      />
    );
  }

  if (currentScreen === 'booking_schedule') {
    return (
      <BookingScheduleScreen
        onNext={() => setCurrentScreen('earnings_overview')}
        onBack={() => setCurrentScreen('onboarding_overview')}
      />
    );
  }

  if (currentScreen === 'earnings_overview') {
    return (
      <EarningsOverviewScreen
        onFinish={() => setCurrentScreen('login')}
        onBack={() => setCurrentScreen('booking_schedule')}
      />
    );
  }

  if (currentScreen === 'login') {
    return (
      <LoginScreen
        onLoginSuccess={() => {
          setCurrentScreen('first_time_setup');
        }}
        onForgotPassword={() => setCurrentScreen('forgot_password')}
        onRegisterLink={() => setCurrentScreen('register')}
      />
    );
  }

  if (currentScreen === 'register') {
    return (
      <RegisterScreen
        onRegisterSuccess={() => {
          setCurrentScreen('first_time_setup');
        }}
        onBackToLogin={() => setCurrentScreen('login')}
      />
    );
  }

  if (currentScreen === 'forgot_password') {
    return (
      <ForgotPasswordScreen
        onBackToLogin={() => setCurrentScreen('login')}
      />
    );
  }

  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <StatusBar style="light" />
        <MainScreen onOpenSetup={() => setCurrentScreen('first_time_setup')} />
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

function MainScreen({ onOpenSetup }: { onOpenSetup?: () => void }) {
  const [activeTab, setActiveTab] = useState<'trips' | 'dispatch'>('trips');
  const [locationState, setLocationState] = useState<{ granted: boolean; latitude: number | null; longitude: number | null; error: string | null }>({
    granted: false,
    latitude: null,
    longitude: null,
    error: null,
  });
  const [cameraState, setCameraState] = useState<{ granted: boolean; error: string | null }>({
    granted: false,
    error: null,
  });

  const { user, loadSession } = useAuthStore();

  useEffect(() => {
    Analytics.init();
    Analytics.identify(user?.id || DEMO_DRIVER_ID, { role: 'fleet_driver' });
    loadSession().then(() => {
      MQTT.connect(user?.id || DEMO_DRIVER_ID);
      SyncEngine.startAutoSync(user?.id || DEMO_DRIVER_ID, 15000);
    });
    return () => SyncEngine.stopAutoSync();
  }, []);

  const handleManualSync = async () => {
    try {
      Analytics.track('driver_manual_sync_clicked');
      const res = await SyncEngine.syncPendingLogs(user?.id || DEMO_DRIVER_ID);
      if (res.error) {
        Alert.alert('Sync Warning', res.error);
      } else {
        Alert.alert('Auto-Sync Engine Success', `Successfully synced ${res.syncedCount} offline GPS records to Go backend.`);
        handleFetchDBLogs();
      }
    } catch (e: any) {
      Alert.alert('Sync Error', e.message || 'Failed to sync');
    }
  };

  const handleRequestLocation = async () => {
    try {
      console.log('Location button clicked');
      Analytics.track('driver_gps_permission_requested');

      const loc = await Telemetry.requestLocationPermission();
      
      const finalLoc = {
        granted: true,
        latitude: loc.latitude || DEMO_LATITUDE,
        longitude: loc.longitude || DEMO_LONGITUDE,
        error: loc.error,
      };

      setLocationState(finalLoc);
      Analytics.track('driver_gps_location_acquired', { lat: finalLoc.latitude, lng: finalLoc.longitude });
      
      // Stream live location over MQTT protocol
      MQTT.publishLocation(user?.id || DEMO_DRIVER_ID, finalLoc.latitude, finalLoc.longitude);

      Alert.alert('GPS Access Granted', `Latitude: ${finalLoc.latitude.toFixed(4)}, Longitude: ${finalLoc.longitude.toFixed(4)}\nStreamed over MQTT & Saved to SQLite.`);
      
      Telemetry.startLiveLocationTracking((lat, lng) => {
        setLocationState((prev) => ({ ...prev, latitude: lat, longitude: lng }));
        MQTT.publishLocation(user?.id || DEMO_DRIVER_ID, lat, lng);
      });
    } catch (e: any) {
      Analytics.track('driver_gps_error', { error: e.message });
      Alert.alert('Location Error', e.message || 'Failed to request location');
    }
  };

  const [showCameraView, setShowCameraView] = useState(false);
  const [dbLogs, setDbLogs] = useState<Array<{ id: number; latitude: number; longitude: number; timestamp: string }>>([]);

  const handleFetchDBLogs = async () => {
    try {
      Analytics.track('driver_fetched_db_logs');
      const logs = await DB.getUnsyncedGPSLogs();
      setDbLogs(logs.slice(-5));
      Alert.alert('SQLite Database Query Success', `Retrieved ${logs.length} persisted location logs from mobile SQLite DB.`);
    } catch (e: any) {
      Alert.alert('DB Error', e.message || 'Failed to read SQLite database');
    }
  };

  const handleRequestCamera = async () => {
    try {
      Analytics.track('driver_camera_requested');
      const cam = await Telemetry.requestCameraPermission();
      setCameraState(cam);
      if (cam.granted) {
        setShowCameraView(true);
        Analytics.track('driver_camera_viewfinder_opened');
      } else {
        Alert.alert('Camera Permission Denied', cam.error || 'Please grant camera permission in Settings.');
      }
    } catch (e: any) {
      Alert.alert('Camera Error', e.message || 'Failed to request camera');
    }
  };

  const { data: trips, isLoading } = useQuery<Trip[]>({
    queryKey: ['trips'],
    queryFn: async () => {
      const mockData: Trip[] = [
        {
          id: '1',
          tripNumber: 'TRP-8492',
          driverName: 'Rajesh Kumar',
          vehiclePlate: 'MH-12-PQ-4521',
          origin: 'Mumbai Central Depot',
          destination: 'Pune Distribution Hub',
          status: 'IN_TRANSIT',
          startTime: '10:30 AM',
        },
        {
          id: '2',
          tripNumber: 'TRP-8493',
          driverName: 'Amit Singh',
          vehiclePlate: 'DL-01-AB-1234',
          origin: 'Delhi Logistics Center',
          destination: 'Jaipur Terminal',
          status: 'PENDING',
          startTime: '11:15 AM',
        },
      ];
      await DB.saveTrips(mockData);
      return await DB.getTrips();
    },
  });

  return (
    <SafeAreaView style={styles.container} edges={['top', 'left', 'right']}>
      <StatusBar style="light" />

      {/* Header with Avandab Brand Colors */}
      <View style={styles.header}>
        <View style={styles.headerTopRow}>
          <View style={styles.brandBadge}>
            <View style={styles.brandDot} />
            <Text style={styles.brandBadgeText}>AVANDAB · OPS</Text>
          </View>
          <Text style={styles.headerClock}>14:32 IST</Text>
        </View>
        <Text style={styles.headerTitle}>FLEET MOBILE</Text>
        <Text style={styles.headerSubtitle}>
          {user ? `${user.name.toUpperCase()} · ${user.id || 'DRV-9042'}` : 'LIVE DISPATCH & TRIP MGMT'}
        </Text>
      </View>

      {/* Non-Blocking Setup Prompt Banner */}
      <View style={styles.bannerContainer}>
        <View style={styles.bannerIconBox}>
          <MaterialCommunityIcons name="clipboard-alert-outline" size={14} color={Colors.warning} />
        </View>
        <View style={styles.bannerTextContainer}>
          <Text style={styles.bannerTitle}>PROFILE SETUP INCOMPLETE</Text>
          <Text style={styles.bannerSub}>Bank details · Profile photo · Driver docs</Text>
        </View>
        <TouchableOpacity
          style={styles.bannerBtn}
          activeOpacity={0.85}
          onPress={() => onOpenSetup && onOpenSetup()}
        >
          <Text style={styles.bannerBtnText}>SETUP</Text>
        </TouchableOpacity>
      </View>

      {/* Tabs */}
      <View style={styles.tabContainer}>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'trips' && styles.activeTab]}
          onPress={() => setActiveTab('trips')}
        >
          <Text style={[styles.tabText, activeTab === 'trips' && styles.activeTabText]}>ACTIVE TRIPS</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'dispatch' && styles.activeTab]}
          onPress={() => setActiveTab('dispatch')}
        >
          <Text style={[styles.tabText, activeTab === 'dispatch' && styles.activeTabText]}>DISPATCH</Text>
        </TouchableOpacity>
      </View>

      {/* Content */}
      <ScrollView style={styles.content} contentContainerStyle={styles.contentPadding}>
        {activeTab === 'trips' ? (
          isLoading ? (
            <>
              <SkeletonLoader />
              <SkeletonLoader />
            </>
          ) : (
            trips?.map((trip) => (
              <TripCard
                key={trip.id}
                tripNumber={trip.tripNumber}
                driverName={trip.driverName}
                vehiclePlate={trip.vehiclePlate}
                origin={trip.origin}
                destination={trip.destination}
                status={trip.status}
                startTime={trip.startTime}
              />
            ))
          )
        ) : (
          <View style={styles.infoCard}>
            <View style={styles.infoCardHeader}>
              <Text style={styles.infoTitle}>TELEMETRY & INSTRUMENTATION</Text>
              <Text style={styles.infoMeta}>DISPATCH PANEL</Text>
            </View>
            <Text style={styles.infoBody}>
              Request native permissions and monitor instrumented GPS location & camera state.
            </Text>

            {/* Telemetry Status Grid */}
            <View style={styles.telemetrySection}>
              <View style={styles.telemetryRow}>
                <Text style={styles.telemetryLabel}>GPS TELEMETRY</Text>
                <View style={[styles.statusPill, locationState.granted ? styles.statusPillActive : styles.statusPillPending]}>
                  <View style={[styles.statusPillDot, { backgroundColor: locationState.granted ? Colors.success : Colors.warning }]} />
                  <Text style={[styles.telemetryValue, { color: locationState.granted ? Colors.success : Colors.warning }]}>
                    {locationState.granted ? 'ACTIVE · 10S' : 'NOT GRANTED'}
                  </Text>
                </View>
              </View>

              {locationState.granted && locationState.latitude ? (
                <View style={styles.gpsDisplayBox}>
                  <View style={styles.gpsRow}>
                    <Text style={styles.gpsLabel}>LAT</Text>
                    <Text style={styles.gpsValue}>{locationState.latitude.toFixed(6)}°N</Text>
                  </View>
                  <View style={styles.gpsRow}>
                    <Text style={styles.gpsLabel}>LNG</Text>
                    <Text style={styles.gpsValue}>{locationState.longitude?.toFixed(6)}°E</Text>
                  </View>
                  <View style={styles.gpsRow}>
                    <Text style={styles.gpsLabel}>PERSIST</Text>
                    <Text style={styles.gpsSuccessText}>SQLITE · MQTT STREAM</Text>
                  </View>

                  {/* Uber-Style Live Interactive Map */}
                  <LiveDriverTrackingMap
                    driverLatitude={locationState.latitude}
                    driverLongitude={locationState.longitude || DEMO_LONGITUDE}
                  />

                  <View style={{ flexDirection: 'row', gap: 8, marginTop: 8 }}>
                    <TouchableOpacity style={[styles.dbFetchBtn, { flex: 1 }]} onPress={handleFetchDBLogs}>
                      <MaterialCommunityIcons name="database-search-outline" size={12} color={Colors.textOnPrimary} />
                      <Text style={styles.dbFetchBtnText}>FETCH LOGS</Text>
                    </TouchableOpacity>

                    <TouchableOpacity style={[styles.dbFetchBtn, styles.dbSyncBtn, { flex: 1 }]} onPress={handleManualSync}>
                      <MaterialCommunityIcons name="cloud-upload-outline" size={12} color={Colors.textOnPrimary} />
                      <Text style={styles.dbFetchBtnText}>SYNC BACKEND</Text>
                    </TouchableOpacity>
                  </View>

                  {dbLogs.length > 0 && (
                    <View style={styles.dbLogsContainer}>
                      <Text style={styles.dbLogsTitle}>OFFLINE_GPS_LOGS · {dbLogs.length} ROWS</Text>
                      {dbLogs.map((log) => (
                        <View key={log.id} style={styles.dbLogRow}>
                          <Text style={styles.dbLogId}>#{log.id}</Text>
                          <Text style={styles.dbLogCoords}>{log.latitude.toFixed(4)}, {log.longitude.toFixed(4)}</Text>
                          <Text style={styles.dbLogTime}>{new Date(log.timestamp).toLocaleTimeString()}</Text>
                        </View>
                      ))}
                    </View>
                  )}
                </View>
              ) : (
                <TouchableOpacity style={styles.actionBtn} onPress={handleRequestLocation}>
                  <MaterialCommunityIcons name="crosshairs-gps" size={14} color={Colors.textOnPrimary} />
                  <Text style={styles.actionBtnText}>REQUEST & INSTRUMENT GPS</Text>
                </TouchableOpacity>
              )}
            </View>

            <View style={styles.divider} />

            <View style={styles.telemetrySection}>
              <View style={styles.telemetryRow}>
                <Text style={styles.telemetryLabel}>CAMERA HARDWARE</Text>
                <View style={[styles.statusPill, cameraState.granted ? styles.statusPillActive : styles.statusPillPending]}>
                  <View style={[styles.statusPillDot, { backgroundColor: cameraState.granted ? Colors.success : Colors.warning }]} />
                  <Text style={[styles.telemetryValue, { color: cameraState.granted ? Colors.success : Colors.warning }]}>
                    {cameraState.granted ? 'READY' : 'NOT GRANTED'}
                  </Text>
                </View>
              </View>

              {showCameraView ? (
                <View style={styles.cameraContainer}>
                  <CameraView style={styles.cameraView} facing="back">
                    <View style={styles.scannerOverlay}>
                      <View style={styles.scanTargetBox} />
                      <Text style={styles.scanInstructionText}>ALIGN CARGO BARCODE</Text>
                    </View>
                  </CameraView>
                  <TouchableOpacity style={styles.closeCameraBtn} onPress={() => setShowCameraView(false)}>
                    <Text style={styles.closeCameraBtnText}>CLOSE FINDER</Text>
                  </TouchableOpacity>
                </View>
              ) : (
                <TouchableOpacity style={[styles.actionBtn, styles.actionBtnTeal]} onPress={handleRequestCamera}>
                  <MaterialCommunityIcons name="barcode-scan" size={14} color={Colors.textOnPrimary} />
                  <Text style={styles.actionBtnText}>OPEN BARCODE SCANNER</Text>
                </TouchableOpacity>
              )}
            </View>
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}



const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  header: {
    backgroundColor: Colors.chrome,
    paddingHorizontal: Spacing.lg,
    paddingTop: Spacing.md,
    paddingBottom: Spacing.lg,
  },
  headerTopRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  brandBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    backgroundColor: Colors.chromeLight,
    borderWidth: 1,
    borderColor: Colors.chromeBorder,
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: Radius.sm,
  },
  brandDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: '#22c55e',
  },
  brandBadgeText: {
    color: Colors.textOnChrome,
    fontSize: 9,
    fontWeight: '800',
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  headerClock: {
    color: Colors.textOnChromeMuted,
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  headerTitle: {
    color: Colors.textOnChrome,
    fontSize: 22,
    fontWeight: '900',
    letterSpacing: 3,
    fontFamily: Font.mono,
  },
  headerSubtitle: {
    color: Colors.textOnChromeMuted,
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 1,
    fontFamily: Font.mono,
    marginTop: 4,
  },
  bannerContainer: {
    backgroundColor: Colors.warningBg,
    borderWidth: 1,
    borderColor: '#fde68a',
    borderLeftWidth: 3,
    borderLeftColor: Colors.warning,
    borderRadius: Radius.md,
    padding: Spacing.md,
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: Spacing.lg,
    gap: Spacing.md,
  },
  bannerIconBox: {
    width: 28,
    height: 28,
    borderRadius: Radius.sm,
    backgroundColor: '#fef3c7',
    alignItems: 'center',
    justifyContent: 'center',
  },
  bannerTextContainer: {
    flex: 1,
  },
  bannerTitle: {
    fontSize: 11,
    fontWeight: '800',
    color: Colors.warning,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  bannerSub: {
    fontSize: 10,
    color: Colors.textSecondary,
    marginTop: 2,
    fontFamily: Font.mono,
  },
  bannerBtn: {
    backgroundColor: Colors.warning,
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: Radius.sm,
  },
  bannerBtnText: {
    color: '#ffffff',
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  tabContainer: {
    flexDirection: 'row',
    backgroundColor: Colors.surface,
    borderBottomWidth: 1,
    borderBottomColor: Colors.border,
  },
  tab: {
    flex: 1,
    paddingVertical: 12,
    alignItems: 'center',
  },
  activeTab: {
    borderBottomWidth: 2,
    borderBottomColor: Colors.primary,
  },
  tabText: {
    fontSize: 11,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  activeTabText: {
    color: Colors.primary,
    fontWeight: '800',
  },
  content: {
    flex: 1,
  },
  contentPadding: {
    padding: Spacing.lg,
  },
  infoCard: {
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    padding: Spacing.md,
    borderWidth: 1,
    borderColor: Colors.border,
  },
  infoCardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 6,
  },
  infoTitle: {
    fontSize: 11,
    fontWeight: '800',
    color: Colors.textPrimary,
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  infoMeta: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  infoBody: {
    fontSize: 12,
    color: Colors.textSecondary,
    lineHeight: 18,
  },
  telemetrySection: {
    marginTop: Spacing.md,
  },
  telemetryRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 6,
  },
  telemetryLabel: {
    fontSize: 10,
    fontWeight: '800',
    color: Colors.textPrimary,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  telemetryValue: {
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 0.5,
    fontFamily: Font.mono,
  },
  statusPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: Radius.sm,
    borderWidth: 1,
  },
  statusPillActive: {
    backgroundColor: Colors.successBg,
    borderColor: '#bbf7d0',
  },
  statusPillPending: {
    backgroundColor: Colors.warningBg,
    borderColor: '#fde68a',
  },
  statusPillDot: {
    width: 5,
    height: 5,
    borderRadius: 2,
  },
  gpsDisplayBox: {
    backgroundColor: Colors.surfaceSecondary,
    borderRadius: Radius.md,
    padding: Spacing.md,
    marginTop: 8,
    borderWidth: 1,
    borderColor: Colors.border,
    gap: 6,
  },
  gpsRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  gpsLabel: {
    fontSize: 10,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  gpsValue: {
    fontSize: 12,
    fontWeight: '800',
    color: Colors.primary,
    fontFamily: Font.mono,
  },
  gpsSuccessText: {
    fontSize: 10,
    fontWeight: '700',
    color: Colors.success,
    letterSpacing: 0.5,
    fontFamily: Font.mono,
  },
  dbFetchBtn: {
    backgroundColor: Colors.info,
    paddingVertical: 8,
    paddingHorizontal: 10,
    borderRadius: Radius.sm,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    marginTop: 6,
  },
  dbSyncBtn: {
    backgroundColor: Colors.primary,
  },
  dbFetchBtnText: {
    color: Colors.textOnPrimary,
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  dbLogsContainer: {
    marginTop: 10,
    paddingTop: 10,
    borderTopWidth: 1,
    borderTopColor: Colors.borderLight,
  },
  dbLogsTitle: {
    fontSize: 9,
    fontWeight: '800',
    color: Colors.textPrimary,
    letterSpacing: 1,
    marginBottom: 6,
    fontFamily: Font.mono,
  },
  dbLogRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 4,
    borderBottomWidth: 1,
    borderBottomColor: Colors.borderLight,
  },
  dbLogId: {
    fontSize: 10,
    fontWeight: '800',
    color: Colors.primary,
    fontFamily: Font.mono,
  },
  dbLogCoords: {
    fontSize: 10,
    fontFamily: Font.mono,
    color: Colors.textPrimary,
  },
  dbLogTime: {
    fontSize: 10,
    color: Colors.textSecondary,
    fontFamily: Font.mono,
  },
  coordsText: {
    fontSize: 12,
    color: Colors.textSecondary,
    marginBottom: 8,
  },
  divider: {
    height: 1,
    backgroundColor: Colors.borderLight,
    marginVertical: Spacing.md,
  },
  actionBtn: {
    backgroundColor: Colors.chrome,
    paddingVertical: 12,
    paddingHorizontal: 14,
    borderRadius: Radius.sm,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    marginTop: 6,
  },
  actionBtnTeal: {
    backgroundColor: Colors.primary,
  },
  actionBtnText: {
    color: Colors.textOnPrimary,
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  cameraContainer: {
    marginTop: 10,
    borderRadius: Radius.md,
    overflow: 'hidden',
  },
  cameraView: {
    height: 220,
    width: '100%',
    justifyContent: 'center',
    alignItems: 'center',
  },
  scannerOverlay: {
    flex: 1,
    width: '100%',
    backgroundColor: 'rgba(0,0,0,0.5)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  scanTargetBox: {
    width: 180,
    height: 120,
    borderWidth: 2,
    borderColor: Colors.primary,
    borderRadius: Radius.sm,
    backgroundColor: 'transparent',
  },
  scanInstructionText: {
    color: '#ffffff',
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 1.5,
    marginTop: 10,
    fontFamily: Font.mono,
  },
  closeCameraBtn: {
    backgroundColor: Colors.danger,
    paddingVertical: 10,
    alignItems: 'center',
  },
  closeCameraBtnText: {
    color: '#ffffff',
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
});
