import React, { useState } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors, Font, Radius, Spacing } from '../constants/theme';
import { useAuthStore } from '../stores/authStore';

interface FirstTimeSetupScreenProps {
  onCompleteSetup: () => void;
  onBack: () => void;
}

export function FirstTimeSetupScreen({ onCompleteSetup, onBack }: FirstTimeSetupScreenProps) {
  const user = useAuthStore((state) => state.user);
  const driverName = user?.name ? user.name.split(' ')[0] : 'ESTHER';

  const [completedSteps, setCompletedSteps] = useState<{ [key: string]: boolean }>({
    profilePicture: false,
    bankDetails: false,
    drivingDetails: false,
    governmentId: true,
  });

  const toggleStep = (stepKey: string, stepTitle: string) => {
    const isCompleted = !completedSteps[stepKey];
    setCompletedSteps((prev) => ({ ...prev, [stepKey]: isCompleted }));
    Alert.alert('Step Updated', `${stepTitle} marked as ${isCompleted ? 'Completed' : 'Pending'}.`);
  };

  const handleContinue = () => {
    const pendingCount = Object.keys(completedSteps).filter(
      (key) => key !== 'governmentId' && !completedSteps[key]
    ).length;

    if (pendingCount > 0) {
      Alert.alert(
        'Setup Incomplete',
        `${pendingCount} required step(s) pending. Proceed anyway for demo mode?`,
        [
          { text: 'Complete Now', style: 'cancel' },
          { text: 'Proceed', onPress: onCompleteSetup },
        ]
      );
    } else {
      onCompleteSetup();
    }
  };

  const totalSteps = Object.keys(completedSteps).length;
  const doneSteps = Object.keys(completedSteps).filter((k) => completedSteps[k]).length;
  const progressPct = Math.round((doneSteps / totalSteps) * 100);

  return (
    <View style={styles.container}>
      <StatusBar style="light" />

      <View style={styles.header}>
        <TouchableOpacity style={styles.iconButton} onPress={onBack}>
          <MaterialCommunityIcons name="arrow-left" size={18} color={Colors.textOnChrome} />
        </TouchableOpacity>
        <Text style={styles.headerLabel}>SETUP · {progressPct}%</Text>
        <View style={{ width: 32 }} />
      </View>

      {/* Progress bar */}
      <View style={styles.progressTrack}>
        <View style={[styles.progressFill, { width: `${progressPct}%` }]} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        <View style={styles.welcomeSection}>
          <Text style={styles.welcomeTitle}>WELCOME, {driverName.toUpperCase()}</Text>
          <View style={styles.titleUnderline} />
          <Text style={styles.welcomeSubtitle}>Complete profile setup to activate dispatch eligibility.</Text>
        </View>

        <View style={styles.section}>
          <View style={styles.sectionHeaderRow}>
            <Text style={styles.sectionHeader}>REQUIRED STEPS</Text>
            <Text style={styles.sectionMeta}>
              {Object.keys(completedSteps).filter((k) => k !== 'governmentId' && completedSteps[k]).length}/3 DONE
            </Text>
          </View>

          <TouchableOpacity
            style={styles.stepCard}
            activeOpacity={0.8}
            onPress={() => toggleStep('profilePicture', 'Profile Picture')}
          >
            <View style={styles.stepLeft}>
              <View style={[styles.stepIconBox, completedSteps.profilePicture && styles.stepIconBoxDone]}>
                <MaterialCommunityIcons
                  name={completedSteps.profilePicture ? 'check' : 'account-box-outline'}
                  size={16}
                  color={completedSteps.profilePicture ? Colors.success : Colors.primary}
                />
              </View>
              <View>
                <Text style={styles.stepTitle}>Profile Picture</Text>
                <Text style={styles.stepMeta}>{completedSteps.profilePicture ? 'UPLOADED' : 'PENDING'}</Text>
              </View>
            </View>
            <MaterialCommunityIcons name="chevron-right" size={16} color={Colors.textMuted} />
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.stepCard}
            activeOpacity={0.8}
            onPress={() => toggleStep('bankDetails', 'Bank Account Details')}
          >
            <View style={styles.stepLeft}>
              <View style={[styles.stepIconBox, completedSteps.bankDetails && styles.stepIconBoxDone]}>
                <MaterialCommunityIcons
                  name={completedSteps.bankDetails ? 'check' : 'bank-outline'}
                  size={16}
                  color={completedSteps.bankDetails ? Colors.success : Colors.primary}
                />
              </View>
              <View>
                <Text style={styles.stepTitle}>Bank Account Details</Text>
                <Text style={styles.stepMeta}>{completedSteps.bankDetails ? 'VERIFIED' : 'PENDING'}</Text>
              </View>
            </View>
            <MaterialCommunityIcons name="chevron-right" size={16} color={Colors.textMuted} />
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.stepCard}
            activeOpacity={0.8}
            onPress={() => toggleStep('drivingDetails', 'Driving Details')}
          >
            <View style={styles.stepLeft}>
              <View style={[styles.stepIconBox, completedSteps.drivingDetails && styles.stepIconBoxDone]}>
                <MaterialCommunityIcons
                  name={completedSteps.drivingDetails ? 'check' : 'card-account-details-outline'}
                  size={16}
                  color={completedSteps.drivingDetails ? Colors.success : Colors.primary}
                />
              </View>
              <View>
                <Text style={styles.stepTitle}>Driving Details</Text>
                <Text style={styles.stepMeta}>{completedSteps.drivingDetails ? 'SUBMITTED' : 'PENDING'}</Text>
              </View>
            </View>
            <MaterialCommunityIcons name="chevron-right" size={16} color={Colors.textMuted} />
          </TouchableOpacity>
        </View>

        <View style={styles.section}>
          <View style={styles.sectionHeaderRow}>
            <Text style={styles.sectionHeader}>SUBMITTED STEPS</Text>
            <Text style={styles.sectionMetaSuccess}>VERIFIED</Text>
          </View>

          <View style={[styles.stepCard, styles.stepCardDone]}>
            <View style={styles.stepLeft}>
              <View style={[styles.stepIconBox, styles.stepIconBoxDone]}>
                <MaterialCommunityIcons name="shield-check-outline" size={16} color={Colors.success} />
              </View>
              <View>
                <Text style={styles.stepTitle}>Government ID</Text>
                <Text style={styles.stepMetaSuccess}>VERIFIED · 2024-08-11</Text>
              </View>
            </View>
            <MaterialCommunityIcons name="lock" size={14} color={Colors.textMuted} />
          </View>
        </View>
      </ScrollView>

      <View style={styles.bottomBar}>
        <TouchableOpacity style={styles.continueBtn} activeOpacity={0.88} onPress={handleContinue}>
          <Text style={styles.continueBtnText}>CONTINUE</Text>
          <MaterialCommunityIcons name="arrow-right" size={14} color={Colors.textOnPrimary} />
        </TouchableOpacity>
      </View>
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
  progressTrack: {
    height: 3,
    backgroundColor: Colors.chromeLight,
  },
  progressFill: {
    height: '100%',
    backgroundColor: Colors.primary,
  },
  scrollContent: {
    paddingHorizontal: Spacing.lg,
    paddingTop: Spacing.xl,
    paddingBottom: 100,
  },
  welcomeSection: {
    marginBottom: Spacing.xl,
  },
  welcomeTitle: {
    fontSize: 18,
    fontWeight: '900',
    color: Colors.textPrimary,
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
  titleUnderline: {
    width: 28,
    height: 2,
    backgroundColor: Colors.primary,
    marginTop: 6,
    marginBottom: Spacing.md,
  },
  welcomeSubtitle: {
    fontSize: 12,
    color: Colors.textSecondary,
    lineHeight: 18,
  },
  section: {
    marginBottom: Spacing.xl,
  },
  sectionHeaderRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: Spacing.md,
  },
  sectionHeader: {
    fontSize: 11,
    fontWeight: '800',
    color: Colors.textPrimary,
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
  sectionMeta: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  sectionMetaSuccess: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.success,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  stepCard: {
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    borderWidth: 1,
    borderColor: Colors.border,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  stepCardDone: {
    backgroundColor: Colors.surfaceSecondary,
    borderColor: Colors.borderLight,
  },
  stepLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  stepIconBox: {
    width: 32,
    height: 32,
    borderRadius: Radius.sm,
    backgroundColor: Colors.primaryLight,
    alignItems: 'center',
    justifyContent: 'center',
  },
  stepIconBoxDone: {
    backgroundColor: Colors.successBg,
  },
  stepTitle: {
    fontSize: 13,
    fontWeight: '700',
    color: Colors.textPrimary,
  },
  stepMeta: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
    marginTop: 2,
  },
  stepMetaSuccess: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.success,
    letterSpacing: 1,
    fontFamily: Font.mono,
    marginTop: 2,
  },
  bottomBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    backgroundColor: Colors.surface,
    paddingHorizontal: Spacing.lg,
    paddingTop: Spacing.md,
    paddingBottom: Spacing.xl,
    borderTopWidth: 1,
    borderColor: Colors.border,
  },
  continueBtn: {
    height: 48,
    backgroundColor: Colors.primary,
    borderRadius: Radius.md,
    alignItems: 'center',
    justifyContent: 'center',
    flexDirection: 'row',
    gap: 8,
  },
  continueBtnText: {
    color: Colors.textOnPrimary,
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
});
