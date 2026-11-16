package com.fudia.pay;

import com.fudia.pay.core.network.FudiaPayApi;
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
public final class MainActivity_MembersInjector implements MembersInjector<MainActivity> {
  private final Provider<DeviceCredentials> credentialsProvider;

  private final Provider<SyncScheduler> schedulerProvider;

  private final Provider<PendingEventDao> daoProvider;

  private final Provider<FudiaPayApi> apiProvider;

  public MainActivity_MembersInjector(Provider<DeviceCredentials> credentialsProvider,
      Provider<SyncScheduler> schedulerProvider, Provider<PendingEventDao> daoProvider,
      Provider<FudiaPayApi> apiProvider) {
    this.credentialsProvider = credentialsProvider;
    this.schedulerProvider = schedulerProvider;
    this.daoProvider = daoProvider;
    this.apiProvider = apiProvider;
  }

  public static MembersInjector<MainActivity> create(
      Provider<DeviceCredentials> credentialsProvider, Provider<SyncScheduler> schedulerProvider,
      Provider<PendingEventDao> daoProvider, Provider<FudiaPayApi> apiProvider) {
    return new MainActivity_MembersInjector(credentialsProvider, schedulerProvider, daoProvider, apiProvider);
  }

  @Override
  public void injectMembers(MainActivity instance) {
    injectCredentials(instance, credentialsProvider.get());
    injectScheduler(instance, schedulerProvider.get());
    injectDao(instance, daoProvider.get());
    injectApi(instance, apiProvider.get());
  }

  @InjectedFieldSignature("com.fudia.pay.MainActivity.credentials")
  public static void injectCredentials(MainActivity instance, DeviceCredentials credentials) {
    instance.credentials = credentials;
  }

  @InjectedFieldSignature("com.fudia.pay.MainActivity.scheduler")
  public static void injectScheduler(MainActivity instance, SyncScheduler scheduler) {
    instance.scheduler = scheduler;
  }

  @InjectedFieldSignature("com.fudia.pay.MainActivity.dao")
  public static void injectDao(MainActivity instance, PendingEventDao dao) {
    instance.dao = dao;
  }

  @InjectedFieldSignature("com.fudia.pay.MainActivity.api")
  public static void injectApi(MainActivity instance, FudiaPayApi api) {
    instance.api = api;
  }
}
