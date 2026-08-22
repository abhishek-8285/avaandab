import React, { useEffect, useState } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors, Font, Radius, Spacing } from '../constants/theme';
import { getApiBaseURL } from '../constants/network';
import { useAuthStore } from '../stores/authStore';
import { MQTT } from '../services/mqtt';
import { BackgroundGPS } from '../services/backgroundLocation';

interface DriverProfile {
  driver_id: string;
  name: string;
  phone: string;
  status: string;
  vehicle_plate: string;
}

interface ProfileScreenProps {
  onBack: () => void;
}

const DUTY_OPTIONS = [
  { id: 'available', label: 'AVAILABLE' },
  { id: 'leave', label: 'ON LEAVE' },
  { id: 'inactive', label: 'OFF DUTY' },
] as const;

export function ProfileScreen({ onBack }: ProfileScreenProps) {
  const { token, user, logout } = useAuthStore();
  const [profile, setProfile] = useState<DriverProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [bgGpsOn, setBgGpsOn] = useState(false);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const res = await fetch(`${getApiBaseURL()}/api/v1/drivers/me`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (res.ok) {
          const json = await res.json();
          if (alive) setProfile(json);
        }
      } catch {
        // offline — fall through to auth-store identity below
      } finally {
        if (alive) {
          setBgGpsOn(await BackgroundGPS.isRunning());
          setLoading(false);
        }
      }
    })();
    return () => {
      alive = false;
    };
  }, [token]);

  const setDutyStatus = async (status: string) => {
    try {
      const res = await fetch(`${getApiBaseURL()}/api/v1/drivers/me/status`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ status }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
      }
      setProfile((prev) => (prev ? { ...prev, status } : prev));
    } catch (e: any) {
      Alert.alert('Failed', e?.message || 'Could not update duty status.');
    }
  };

  const handleSignOut = () => {
    Alert.alert('Sign Out', 'End this session on the device?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign Out',
        style: 'destructive',
        onPress: () => {
          MQTT.disconnect();
          BackgroundGPS.stop();
          logout();
        },
      },
    ]);
  };

  const displayName = profile?.name || user?.name || 'Driver';
  const rows: { label: string; value: string }[] = [
    { label: 'DRIVER ID', value: profile?.driver_id || user?.driverId || user?.id || '—' },
    { label: 'PHONE', value: profile?.phone || '—' },
    { label: 'VEHICLE', value: profile?.vehicle_plate || 'Not assigned yet — contact dispatch.' },
    { label: 'DUTY STATUS', value: (profile?.status || 'unknown').toUpperCase() },
  ];

  return (
    <View style={styles.container}>
      <StatusBar style="light" />

      <View style={styles.header}>
        <TouchableOpacity style={styles.iconButton} onPress={onBack}>
          <MaterialCommunityIcons name="arrow-left" size={18} color={Colors.textOnChrome} />
        </TouchableOpacity>
        <Text style={styles.headerLabel}>PROFILE</Text>
        <View style={{ width: 32 }} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        {loading ? (
          <ActivityIndicator color={Colors.primary} style={{ marginTop: Spacing.xl }} />
        ) : (
          <>
            <View style={styles.heroCard}>
              <View style={styles.avatarBox}>
                <MaterialCommunityIcons name="account" size={30} color={Colors.textOnPrimary} />
              </View>
              <Text style={styles.heroName}>{displayName.toUpperCase()}</Text>
              <Text style={styles.heroSub}>{user?.email || ''}</Text>
            </View>

            <View style={styles.card}>
              {rows.map((row) => (
                <View key={row.label} style={styles.row}>
                  <Text style={styles.rowLabel}>{row.label}</Text>
                  <Text style={styles.rowValue}>{row.value}</Text>
                </View>
              ))}
            </View>

            <View style={styles.card}>
              <View style={styles.row}>
                <Text style={styles.rowLabel}>BACKGROUND GPS</Text>
                <Text style={[styles.rowValue, { color: bgGpsOn ? Colors.success : Colors.textMuted }]}>
                  {bgGpsOn ? 'ON · OS-LEVEL' : 'OFF'}
                </Text>
              </View>
              <Text style={styles.hint}>Toggle lives in DISPATCH tab → telemetry panel.</Text>
            </View>

            <TouchableOpacity style={styles.signOutBtn} onPress={handleSignOut}>
              <MaterialCommunityIcons name="logout" size={16} color={Colors.danger} />
              <Text style={styles.signOutText}>SIGN OUT OF DEVICE</Text>
            </TouchableOpacity>

            <Text style={styles.version}>Avandab Fleet Mobile</Text>
          </>
        )}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: Spacing.lg,
    paddingTop: 50,
    paddingBottom: Spacing.md,
    backgroundColor: Colors.chrome,
  },
  headerLabel: {
    fontSize: 11,
    fontWeight: '700',
    color: Colors.textOnChrome,
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
  iconButton: {
    width: 32,
    height: 32,
    borderRadius: Radius.md,
    borderWidth: 1,
    borderColor: Colors.chromeBorder,
    alignItems: 'center',
    justifyContent: 'center',
  },
  scrollContent: {
    padding: Spacing.lg,
    gap: Spacing.md,
  },
  heroCard: {
    backgroundColor: Colors.surface,
    borderRadius: Radius.lg,
    padding: Spacing.xl,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: Colors.border,
  },
  avatarBox: {
    width: 60,
    height: 60,
    borderRadius: 30,
    backgroundColor: Colors.primary,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: Spacing.sm,
  },
  heroName: {
    fontSize: 17,
    fontWeight: '900',
    color: Colors.textPrimary,
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  heroSub: {
    fontSize: 11,
    color: Colors.textSecondary,
    marginTop: 4,
    fontFamily: Font.mono,
  },
  card: {
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    padding: Spacing.lg,
    borderWidth: 1,
    borderColor: Colors.borderLight,
    gap: 10,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: Spacing.md,
  },
  rowLabel: {
    fontSize: 10,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
    minWidth: 110,
  },
  rowValue: {
    flex: 1,
    fontSize: 12,
    fontWeight: '600',
    color: Colors.textPrimary,
    textAlign: 'right',
    fontFamily: Font.mono,
  },
  hint: {
    fontSize: 10,
    color: Colors.textMuted,
    fontStyle: 'italic',
  },
  signOutBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    paddingVertical: 14,
    borderRadius: Radius.md,
    borderWidth: 1,
    borderColor: Colors.danger,
    backgroundColor: Colors.surface,
  },
  signOutText: {
    color: Colors.danger,
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  dutyRow: {
    flex: 1,
    flexDirection: 'row',
    gap: 6,
    justifyContent: 'flex-end',
    flexWrap: 'wrap',
  },
  dutyChip: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: Colors.border,
    backgroundColor: Colors.surfaceSecondary,
  },
  dutyChipActive: {
    backgroundColor: Colors.primary,
    borderColor: Colors.primary,
  },
  dutyChipText: {
    fontSize: 9,
    fontWeight: '800',
    letterSpacing: 0.5,
    color: Colors.textSecondary,
    fontFamily: Font.mono,
  },
  dutyChipTextActive: {
    color: Colors.textOnPrimary,
  },
  version: {
    textAlign: 'center',
    fontSize: 9,
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
    marginBottom: Spacing.lg,
  },
});
