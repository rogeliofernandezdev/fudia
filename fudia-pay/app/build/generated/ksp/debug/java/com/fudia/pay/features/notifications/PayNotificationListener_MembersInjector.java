package com.fudia.pay.features.notifications;

import com.fudia.pay.core.security.DeviceCredentials;
import com.fudia.pay.features.payments.PendingEventDao;
import com.fudia.pay.features.sync.SyncScheduler;
import dagger.MembersInjector;
import dagger.internal.DaggerGenerated;
import dagger.internal.InjectedFieldSignature;
import dagger.internal.QualifierMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

@QualifierMetadata
@DaggerGenerated
@Generated(
    value = "dagger.internal.codegen.ComponentProcessor",
    comments = "https://dagger.dev"
)
@SuppressWarnings({
    "unchecked",
    "rawtypes",
    "KotlinInternal",
    "KotlinInternalInJava",
    "cast",
    "deprecation"
})
public final class PayNotificationListener_MembersInjector implements MembersInjector<PayNotificationListener> {
  private final Provider<DeviceCredentials> credentialsProvider;

  private final Provider<PendingEventDao> daoProvider;

  private final Provider<SyncScheduler> schedulerProvider;

  public PayNotificationListener_MembersInjector(Provider<DeviceCredentials> credentialsProvider,
      Provider<PendingEventDao> daoProvider, Provider<SyncScheduler> schedulerProvider) {
    this.credentialsProvider = credentialsProvider;
    this.daoProvider = daoProvider;
    this.schedulerProvider = schedulerProvider;
  }

  public static MembersInjector<PayNotificationListener> create(
      Provider<DeviceCredentials> credentialsProvider, Provider<PendingEventDao> daoProvider,
      Provider<SyncScheduler> schedulerProvider) {
    return new PayNotificationListener_MembersInjector(credentialsProvider, daoProvider, schedulerProvider);
  }

  @Override
  public void injectMembers(PayNotificationListener instance) {
    injectCredentials(instance, credentialsProvider.get());
    injectDao(instance, daoProvider.get());
    injectScheduler(instance, schedulerProvider.get());
  }

  @InjectedFieldSignature("com.fudia.pay.features.notifications.PayNotificationListener.credentials")
  public static void injectCredentials(PayNotificationListener instance,
      DeviceCredentials credentials) {
    instance.credentials = credentials;
  }

  @InjectedFieldSignature("com.fudia.pay.features.notifications.PayNotificationListener.dao")
  public static void injectDao(PayNotificationListener instance, PendingEventDao dao) {
    instance.dao = dao;
  }

  @InjectedFieldSignature("com.fudia.pay.features.notifications.PayNotificationListener.scheduler")
  public static void injectScheduler(PayNotificationListener instance, SyncScheduler scheduler) {
    instance.scheduler = scheduler;
  }
}
