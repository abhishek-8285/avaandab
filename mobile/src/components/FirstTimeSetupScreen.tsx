import React, { useState } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors } from '../constants/theme';
import { useAuthStore } from '../stores/authStore';

interface FirstTimeSetupScreenProps {
  onCompleteSetup: () => void;
  onBack: () => void;
}

export function FirstTimeSetupScreen({ onCompleteSetup, onBack }: FirstTimeSetupScreenProps) {
  const user = useAuthStore((state) => state.user);
  const driverName = user?.name ? user.name.split(' ')[0] : 'Esther';

  // Track completed/submitted steps
  const [completedSteps, setCompletedSteps] = useState<{ [key: string]: boolean }>({
    profilePicture: false,
    bankDetails: false,
    drivingDetails: false,
    governmentId: true, // Submitted as shown in design mockup
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
        `You still have ${pendingCount} required step(s) pending. Would you like to proceed anyway for demo mode?`,
        [
          { text: 'Complete Now', style: 'cancel' },
          { text: 'Proceed', onPress: onCompleteSetup },
        ]
      );
    } else {
      onCompleteSetup();
    }
  };

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />

      {/* Top Bar with Back Arrow */}
      <View style={styles.header}>
        <TouchableOpacity style={styles.backButton} onPress={onBack}>
          <MaterialCommunityIcons name="arrow-left" size={22} color={Colors.textPrimary} />
        </TouchableOpacity>
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        {/* Welcome Header */}
        <View style={styles.welcomeSection}>
          <Text style={styles.welcomeTitle}>Welcome!, {driverName}</Text>
        </View>

        {/* Required Steps Section */}
        <View style={styles.section}>
          <Text style={styles.sectionHeader}>Required Steps</Text>

          {/* Profile Picture Item */}
          <TouchableOpacity
            style={styles.stepCard}
            activeOpacity={0.8}
            onPress={() => toggleStep('profilePicture', 'Profile Picture')}
          >
            <View style={styles.stepLeft}>
              <MaterialCommunityIcons
                name={completedSteps.profilePicture ? 'check-circle' : 'account-box-outline'}
                size={22}
                color={completedSteps.profilePicture ? Colors.success : Colors.primary}
              />
              <Text style={styles.stepTitle}>Profile Picture</Text>
            </View>
            <MaterialCommunityIcons name="chevron-right" size={24} color={Colors.primary} />
          </TouchableOpacity>

          {/* Bank Account Details Item */}
          <TouchableOpacity
            style={styles.stepCard}
            activeOpacity={0.8}
            onPress={() => toggleStep('bankDetails', 'Bank Account Details')}
          >
            <View style={styles.stepLeft}>
              <MaterialCommunityIcons
                name={completedSteps.bankDetails ? 'check-circle' : 'bank-outline'}
                size={22}
                color={completedSteps.bankDetails ? Colors.success : Colors.primary}
              />
              <Text style={styles.stepTitle}>Bank Account Details</Text>
            </View>
            <MaterialCommunityIcons name="chevron-right" size={24} color={Colors.primary} />
          </TouchableOpacity>

          {/* Driving Details Item */}
          <TouchableOpacity
            style={styles.stepCard}
            activeOpacity={0.8}
            onPress={() => toggleStep('drivingDetails', 'Driving Details')}
          >
            <View style={styles.stepLeft}>
              <MaterialCommunityIcons
                name={completedSteps.drivingDetails ? 'check-circle' : 'card-account-details-outline'}
                size={22}
                color={completedSteps.drivingDetails ? Colors.success : Colors.primary}
              />
              <Text style={styles.stepTitle}>Driving Details</Text>
            </View>
            <MaterialCommunityIcons name="chevron-right" size={24} color={Colors.primary} />
          </TouchableOpacity>
        </View>

        {/* Submitted Steps Section */}
        <View style={styles.section}>
          <Text style={styles.sectionHeader}>Submitted Steps</Text>

          {/* Government ID Item */}
          <TouchableOpacity
            style={styles.stepCard}
            activeOpacity={0.8}
            onPress={() => toggleStep('governmentId', 'Government ID')}
          >
            <View style={styles.stepLeft}>
              <MaterialCommunityIcons name="shield-check-outline" size={22} color={Colors.success} />
              <Text style={styles.stepTitle}>Government ID</Text>
            </View>
            <MaterialCommunityIcons name="chevron-right" size={24} color={Colors.primary} />
          </TouchableOpacity>
        </View>
      </ScrollView>

      {/* Bottom Sticky Action Area */}
      <View style={styles.bottomBar}>
        <TouchableOpacity style={styles.continueBtn} activeOpacity={0.88} onPress={handleContinue}>
          <Text style={styles.continueBtnText}>Continue</Text>
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
    paddingHorizontal: 20,
    paddingTop: 50,
    paddingBottom: 10,
  },
  backButton: {
    width: 44,
    height: 44,
    borderRadius: 22,
    borderWidth: 1,
    borderColor: Colors.border,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: Colors.surface,
  },
  scrollContent: {
    paddingHorizontal: 24,
    paddingTop: 16,
    paddingBottom: 100,
  },
  welcomeSection: {
    alignItems: 'center',
    marginVertical: 24,
  },
  welcomeTitle: {
    fontSize: 28,
    fontWeight: '700',
    color: Colors.textPrimary,
    textAlign: 'center',
  },
  section: {
    marginBottom: 28,
  },
  sectionHeader: {
    fontSize: 16,
    fontWeight: '700',
    color: Colors.textPrimary,
    marginBottom: 14,
  },
  stepCard: {
    backgroundColor: Colors.surface,
    borderRadius: 14,
    paddingHorizontal: 18,
    paddingVertical: 18,
    borderWidth: 1,
    borderColor: Colors.border,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
    shadowColor: Colors.textPrimary,
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.04,
    shadowRadius: 6,
    elevation: 2,
  },
  stepLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  stepTitle: {
    fontSize: 15,
    fontWeight: '600',
    color: Colors.textPrimary,
  },
  bottomBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    backgroundColor: Colors.surface,
    paddingHorizontal: 24,
    paddingTop: 12,
    paddingBottom: 28,
    borderTopWidth: 1,
    borderColor: Colors.border,
  },
  continueBtn: {
    height: 52,
    backgroundColor: Colors.primary,
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.25,
    shadowRadius: 8,
    elevation: 4,
  },
  continueBtnText: {
    color: Colors.textOnPrimary,
    fontSize: 16,
    fontWeight: '700',
  },
});
