package tech.shatrov.familyfinances

import android.app.Application
import tech.shatrov.familyfinances.ui.recognize.ImportFiles

/**
 * Граф живёт на процессе, а не на активити: поворот экрана пересоздаёт активити, и граф из
 * `onCreate` обнулял бы сессию на каждый поворот, а пережившие его ViewModel остались бы с
 * прошлым клиентом — их `401` уже никто бы не слушал.
 */
class FamilyFinancesApp : Application() {
    val graph: AppGraph by lazy { AppGraph.create(this, BuildConfig.API_BASE_URL) }

    // Синхронно: share холодного старта кладёт файлы следом, фоновая чистка стёрла бы их.
    override fun onCreate() {
        super.onCreate()
        ImportFiles.sweep(this)
    }
}
