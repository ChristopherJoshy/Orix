package com.orixridesharestudents.app

import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import android.content.Intent
import android.content.ActivityNotFoundException
import android.net.Uri

class MainActivity : FlutterActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "orix/donations")
            .setMethodCallHandler { call, result ->
                if (call.method != "openGooglePay") {
                    result.notImplemented()
                } else {
                    val uri = (call.arguments as? String)?.let { Uri.parse(it) }
                    if (uri?.scheme != "upi" || uri.host != "pay") {
                        result.error("invalid_payment_link", "Invalid payment link", null)
                    } else {
                        try {
                            startActivity(Intent(Intent.ACTION_VIEW, uri)
                                .setPackage("com.google.android.apps.nbu.paisa.user"))
                            result.success(true)
                        } catch (_: ActivityNotFoundException) {
                            result.success(false)
                        } catch (_: SecurityException) {
                            result.success(false)
                        }
                    }
                }
            }
    }
}
