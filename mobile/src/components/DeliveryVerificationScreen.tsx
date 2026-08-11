import React, { useState } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, Image, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { CameraView, CameraType, useCameraPermissions } from 'expo-camera';
import { Colors } from '../constants/theme';

interface DeliveryVerificationScreenProps {
  onComplete: () => void;
  onBack: () => void;
}

export function DeliveryVerificationScreen({ onComplete, onBack }: DeliveryVerificationScreenProps) {
  const [permission, requestPermission] = useCameraPermissions();
  const [cameraActive, setCameraActive] = useState(false);
  const [capturedPhoto, setCapturedPhoto] = useState<string | null>(null);
  const [cameraRef, setCameraRef] = useState<any>(null);

  const takePhoto = async () => {
    if (cameraRef) {
      try {
        const photo = await cameraRef.takePictureAsync();
        setCapturedPhoto(photo.uri);
        setCameraActive(false);
      } catch (err) {
        Alert.alert('Camera Error', 'Failed to capture photo proof.');
      }
    }
  };

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />

      {/* Top Navigation Bar */}
      <View style={styles.header}>
        <TouchableOpacity style={styles.iconButton} onPress={onBack}>
          <MaterialCommunityIcons name="arrow-left" size={22} color="#0b1c30" />
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Avandab Logistics</Text>
        <TouchableOpacity style={styles.iconButton}>
          <MaterialCommunityIcons name="bell-outline" size={20} color="#0b1c30" />
        </TouchableOpacity>
      </View>

      {cameraActive ? (
        <View style={styles.cameraContainer}>
          {!permission?.granted ? (
            <View style={styles.permissionBox}>
              <Text style={styles.permissionText}>Camera permission is required to capture proof of delivery.</Text>
              <TouchableOpacity style={styles.permissionBtn} onPress={requestPermission}>
                <Text style={styles.permissionBtnText}>Grant Camera Permission</Text>
              </TouchableOpacity>
              <TouchableOpacity style={styles.cancelCameraBtn} onPress={() => setCameraActive(false)}>
                <Text style={styles.cancelCameraText}>Cancel</Text>
              </TouchableOpacity>
            </View>
          ) : (
            <CameraView style={styles.cameraView} ref={(ref) => setCameraRef(ref)}>
              <View style={styles.cameraOverlay}>
                <Text style={styles.cameraGuideText}>Position cargo barcode or item inside frame</Text>
                <TouchableOpacity style={styles.captureBtn} onPress={takePhoto}>
                  <View style={styles.captureInnerCircle} />
                </TouchableOpacity>
                <TouchableOpacity style={styles.closeCameraBtn} onPress={() => setCameraActive(false)}>
                  <MaterialCommunityIcons name="close" size={26} color="#ffffff" />
                </TouchableOpacity>
              </View>
            </CameraView>
          )}
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
          {/* Header Section */}
          <View style={styles.titleSection}>
            <Text style={styles.title}>Complete Delivery</Text>
            <Text style={styles.subtitle}>Order #ORD-7492-X</Text>
          </View>

          {/* Delivery Summary Card */}
          <View style={styles.card}>
            <Text style={styles.cardHeader}>Delivery Summary</Text>
            <View style={styles.summaryItem}>
              <MaterialCommunityIcons name="package-variant-closed" size={24} color={Colors.primary} style={styles.itemIcon} />
              <View style={styles.itemTextContainer}>
                <Text style={styles.itemTitle}>Medical Equipment Box</Text>
                <Text style={styles.itemSubtitle}>Qty: 2 • Fragile • Handle with care</Text>
              </View>
            </View>

            <View style={styles.divider} />

            <View style={styles.summaryItem}>
              <MaterialCommunityIcons name="map-marker-outline" size={24} color={Colors.primary} style={styles.itemIcon} />
              <View style={styles.itemTextContainer}>
                <Text style={styles.itemTitle}>Apollo Medical Center</Text>
                <Text style={styles.itemSubtitle}>Gate 3, Receiving Dock, Mumbai</Text>
              </View>
            </View>
          </View>

          {/* Photo Proof of Delivery Section */}
          <View style={styles.card}>
            <Text style={styles.cardHeader}>Photo Proof of Delivery</Text>
            <Text style={styles.cardSubtitle}>
              Please capture a clear photo of the delivered package at the drop-off location or barcode label.
            </Text>

            {capturedPhoto ? (
              <View style={styles.photoPreviewContainer}>
                <Image source={{ uri: capturedPhoto }} style={styles.photoPreview} />
                <TouchableOpacity style={styles.retakeBtn} onPress={() => setCameraActive(true)}>
                  <MaterialCommunityIcons name="camera-retake-outline" size={18} color="#ffffff" />
                  <Text style={styles.retakeBtnText}>Retake Photo</Text>
                </TouchableOpacity>
              </View>
            ) : (
              <TouchableOpacity
                style={styles.photoPlaceholder}
                activeOpacity={0.8}
                onPress={() => {
                  if (!permission?.granted) {
                    requestPermission().then((res) => {
                      if (res.granted) setCameraActive(true);
                    });
                  } else {
                    setCameraActive(true);
                  }
                }}
              >
                <MaterialCommunityIcons name="camera-plus-outline" size={36} color={Colors.primary} />
                <Text style={styles.placeholderTitle}>Tap to Capture Delivery Photo</Text>
                <Text style={styles.placeholderSub}>Uses device camera for barcode & cargo proof</Text>
              </TouchableOpacity>
            )}
          </View>

          {/* Submit Verification Button */}
          <TouchableOpacity
            style={styles.submitBtn}
            activeOpacity={0.88}
            onPress={() => {
              Alert.alert('Delivery Verified', 'Delivery verification proof submitted successfully!', [
                { text: 'OK', onPress: onComplete },
              ]);
            }}
          >
            <Text style={styles.submitBtnText}>Confirm Delivery Completion</Text>
            <MaterialCommunityIcons name="check-circle-outline" size={20} color="#ffffff" />
          </TouchableOpacity>
        </ScrollView>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f8f9ff',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingTop: 50,
    paddingBottom: 16,
    backgroundColor: '#ffffff',
    borderBottomWidth: 1,
    borderColor: '#e2e8f0',
  },
  iconButton: {
    width: 38,
    height: 38,
    borderRadius: 12,
    backgroundColor: '#eff4ff',
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: Colors.primary,
  },
  scrollContent: {
    paddingHorizontal: 20,
    paddingTop: 20,
    paddingBottom: 40,
  },
  titleSection: {
    marginBottom: 18,
  },
  title: {
    fontSize: 26,
    fontWeight: '700',
    color: '#0b1c30',
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 14,
    color: '#5c647a',
    fontWeight: '500',
  },
  card: {
    backgroundColor: '#ffffff',
    borderRadius: 16,
    padding: 20,
    borderWidth: 1,
    borderColor: '#e2e8f0',
    marginBottom: 16,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 3 },
    shadowOpacity: 0.03,
    shadowRadius: 8,
    elevation: 2,
  },
  cardHeader: {
    fontSize: 16,
    fontWeight: '700',
    color: '#0b1c30',
    marginBottom: 8,
  },
  cardSubtitle: {
    fontSize: 13,
    color: '#5c647a',
    lineHeight: 18,
    marginBottom: 16,
  },
  summaryItem: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  itemIcon: {
    marginRight: 14,
  },
  itemTextContainer: {
    flex: 1,
  },
  itemTitle: {
    fontSize: 15,
    fontWeight: '600',
    color: '#0b1c30',
  },
  itemSubtitle: {
    fontSize: 13,
    color: '#5c647a',
    marginTop: 2,
  },
  divider: {
    height: 1,
    backgroundColor: '#e2e8f0',
    marginVertical: 14,
  },
  photoPlaceholder: {
    height: 160,
    borderWidth: 2,
    borderColor: '#bcc9c6',
    borderStyle: 'dashed',
    borderRadius: 12,
    backgroundColor: '#f4fffc',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 16,
  },
  placeholderTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: Colors.primary,
    marginTop: 8,
  },
  placeholderSub: {
    fontSize: 12,
    color: '#6d7a77',
    marginTop: 4,
  },
  photoPreviewContainer: {
    position: 'relative',
    borderRadius: 12,
    overflow: 'hidden',
  },
  photoPreview: {
    width: '100%',
    height: 220,
    borderRadius: 12,
  },
  retakeBtn: {
    position: 'absolute',
    bottom: 12,
    right: 12,
    backgroundColor: 'rgba(0, 106, 97, 0.9)',
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 8,
  },
  retakeBtnText: {
    color: '#ffffff',
    fontSize: 13,
    fontWeight: '600',
  },
  submitBtn: {
    height: 52,
    backgroundColor: Colors.primary,
    borderRadius: 12,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    marginTop: 8,
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.25,
    shadowRadius: 8,
    elevation: 4,
  },
  submitBtnText: {
    color: '#ffffff',
    fontSize: 16,
    fontWeight: '600',
  },
  cameraContainer: {
    flex: 1,
    backgroundColor: '#000000',
  },
  cameraView: {
    flex: 1,
  },
  cameraOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.3)',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 40,
    paddingHorizontal: 20,
  },
  cameraGuideText: {
    color: '#ffffff',
    fontSize: 14,
    fontWeight: '600',
    backgroundColor: 'rgba(0,0,0,0.6)',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 20,
    marginTop: 40,
  },
  captureBtn: {
    width: 72,
    height: 72,
    borderRadius: 36,
    borderWidth: 4,
    borderColor: '#ffffff',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 20,
  },
  captureInnerCircle: {
    width: 54,
    height: 54,
    borderRadius: 27,
    backgroundColor: '#ffffff',
  },
  closeCameraBtn: {
    position: 'absolute',
    top: 40,
    right: 20,
    padding: 10,
  },
  permissionBox: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 30,
  },
  permissionText: {
    color: '#ffffff',
    fontSize: 16,
    textAlign: 'center',
    marginBottom: 20,
    lineHeight: 22,
  },
  permissionBtn: {
    backgroundColor: Colors.primary,
    paddingHorizontal: 20,
    paddingVertical: 12,
    borderRadius: 8,
    marginBottom: 12,
  },
  permissionBtnText: {
    color: '#ffffff',
    fontWeight: '600',
    fontSize: 14,
  },
  cancelCameraBtn: {
    padding: 10,
  },
  cancelCameraText: {
    color: '#bcc9c6',
    fontSize: 14,
  },
});
