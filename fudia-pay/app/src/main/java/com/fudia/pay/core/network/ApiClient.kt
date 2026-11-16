package com.fudia.pay.core.network

import com.fudia.pay.BuildConfig
import com.fudia.pay.core.security.DeviceCredentials
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import javax.inject.Inject
import javax.inject.Singleton

/** La URL del backend no se configura en la app: el despliegue la fija en
 * BuildConfig (FOODS_API_URL) y el token viaja como credencial del dispositivo. */
class AuthInterceptor @Inject constructor(private val credentials: DeviceCredentials) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): okhttp3.Response {
        val request = chain.request().newBuilder()
            .header("Authorization", "Bearer ${credentials.token}")
            .build()
        return chain.proceed(request)
    }
}

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {
    @Provides
    @Singleton
    fun provideApi(credentials: DeviceCredentials): FudiaPayApi {
        val client = OkHttpClient.Builder()
            .addInterceptor(AuthInterceptor(credentials))
            .build()
        val json = Json { ignoreUnknownKeys = true }
        return Retrofit.Builder()
            .baseUrl(BuildConfig.FOODS_API_URL)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(FudiaPayApi::class.java)
    }
}
