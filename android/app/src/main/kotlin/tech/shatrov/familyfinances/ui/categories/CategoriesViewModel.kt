package tech.shatrov.familyfinances.ui.categories

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.core.api.CreateCategoryRequest
import tech.shatrov.familyfinances.core.api.UpdateCategoryRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import java.util.UUID

/** Справочник берётся одной страницей: категорий у семьи считанные единицы. */
private const val CATEGORY_LIMIT = 200

/** Цвет обязателен и проверяется как `hexcolor`, поэтому выбирается из готового набора. */
val CategoryPalette = listOf(
    "#E53935",
    "#FB8C00",
    "#FDD835",
    "#43A047",
    "#00ACC1",
    "#1E88E5",
    "#8E24AA",
    "#6D4C41",
)

private const val DEFAULT_ICON = "tag"

/** Нижняя граница контракта для `name`. */
private const val MIN_NAME = 2

/** Имена полей формы — те же, что в `error.details[].field`. */
object CategoryField {
    const val NAME = "name"
    const val TYPE = "type"
    const val COLOR = "color"
    const val ICON = "icon"
    const val PARENT = "parent_id"
}

private val formFields = setOf(
    CategoryField.NAME,
    CategoryField.TYPE,
    CategoryField.COLOR,
    CategoryField.ICON,
    CategoryField.PARENT,
)

/** Категория верхнего уровня со своими подкатегориями: вложенность в схеме ровно одна. */
data class CategoryNode(
    val category: Category,
    val children: List<Category>,
)

sealed interface CategoriesUiState {
    data object Loading : CategoriesUiState

    data class Failure(val error: UiError) : CategoriesUiState

    data class Ready(
        val income: List<CategoryNode>,
        val expense: List<CategoryNode>,
    ) : CategoriesUiState {
        val isEmpty: Boolean get() = income.isEmpty() && expense.isEmpty()
    }
}

/**
 * Форма категории. Правка меняет только `name`/`color`/`icon`: тип и родителя сервер
 * после создания не принимает, поэтому на экране правки их нет.
 */
data class CategoryEditUiState(
    val id: UUID? = null,
    val draft: UUID = UUID.randomUUID(),
    val name: String = "",
    val type: CategoryType = CategoryType.expense,
    val color: String = CategoryPalette.first(),
    val icon: String = DEFAULT_ICON,
    val parentId: UUID? = null,
    val parents: List<Category> = emptyList(),
    val canDelete: Boolean = false,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
) {
    val editing: Boolean get() = id != null

    val canSubmit: Boolean get() = name.trim().length >= MIN_NAME && icon.isNotBlank() && !submitting
}

/**
 * Категории: список с формой создания и правки. Удаление показывается только админу —
 * роль известна из сессии, и ловить на неё `403` незачем.
 */
class CategoriesViewModel(
    private val api: ApiGraph,
    private val isAdmin: Boolean,
) : ViewModel() {
    private val mutable = MutableStateFlow<CategoriesUiState>(CategoriesUiState.Loading)
    private val mutableEditor = MutableStateFlow<CategoryEditUiState?>(null)

    val state: StateFlow<CategoriesUiState> = mutable.asStateFlow()

    /** `null` — открыт список; иначе поверх него форма. */
    val editor: StateFlow<CategoryEditUiState?> = mutableEditor.asStateFlow()

    private var loaded = emptyList<Category>()

    init {
        refresh()
    }

    fun refresh() {
        mutable.value = CategoriesUiState.Loading
        viewModelScope.launch {
            try {
                loaded = api.client.unwrap { api.categories.listCategories(limit = CATEGORY_LIMIT) }.`data`
                mutable.value = ready()
            } catch (failure: ApiFailure) {
                mutable.value = CategoriesUiState.Failure(failure.toUiError())
            }
        }
    }

    fun onAdd() {
        mutableEditor.value = CategoryEditUiState(parents = parentsOf(CategoryType.expense, null))
    }

    fun onOpen(category: Category) {
        mutableEditor.value = CategoryEditUiState(
            id = category.id,
            name = category.name,
            type = category.type,
            color = category.color,
            icon = category.icon,
            parentId = category.parentId,
            canDelete = isAdmin,
        )
    }

    fun onDismiss() {
        mutableEditor.value = null
    }

    fun onNameChange(name: String) {
        mutableEditor.update { it?.copy(name = name)?.cleared(CategoryField.NAME) }
    }

    /** Смена типа снимает родителя: под другим типом его в списке нет. */
    fun onTypeChange(type: CategoryType) {
        mutableEditor.update { current ->
            current
                ?.copy(type = type, parentId = null, parents = parentsOf(type, current.id))
                ?.cleared(CategoryField.TYPE)
        }
    }

    fun onColorChange(color: String) {
        mutableEditor.update { it?.copy(color = color)?.cleared(CategoryField.COLOR) }
    }

    fun onIconChange(icon: String) {
        mutableEditor.update { it?.copy(icon = icon)?.cleared(CategoryField.ICON) }
    }

    fun onParentChange(parentId: UUID?) {
        mutableEditor.update { it?.copy(parentId = parentId)?.cleared(CategoryField.PARENT) }
    }

    fun onSubmit() {
        val current = mutableEditor.value ?: return
        if (!current.canSubmit) return
        mutableEditor.value = current.copy(submitting = true, error = null, fieldErrors = emptyMap())
        viewModelScope.launch {
            try {
                save(current)
                mutableEditor.value = null
                refresh()
            } catch (failure: ApiFailure) {
                mutableEditor.update { it?.failed(failure) }
            }
        }
    }

    fun onDelete() {
        val current = mutableEditor.value ?: return
        val id = current.id?.takeIf { current.canDelete } ?: return
        mutableEditor.value = current.copy(submitting = true, error = null)
        viewModelScope.launch {
            try {
                api.client.send { api.categories.deleteCategory(id) }
                mutableEditor.value = null
                refresh()
            } catch (failure: ApiFailure) {
                mutableEditor.update { it?.failed(failure) }
            }
        }
    }

    private suspend fun save(current: CategoryEditUiState) {
        val name = current.name.trim()
        val icon = current.icon.trim()
        val id = current.id
        if (id == null) {
            api.client.unwrap {
                api.categories.createCategory(
                    CreateCategoryRequest(
                        name = name,
                        type = current.type,
                        color = current.color,
                        icon = icon,
                        id = current.draft,
                        parentId = current.parentId,
                    ),
                )
            }
        } else {
            api.client.unwrap {
                api.categories.updateCategory(
                    id,
                    UpdateCategoryRequest(name = name, color = current.color, icon = icon),
                )
            }
        }
    }

    /** Родителем бывает только корневая категория того же типа и не сама запись. */
    private fun parentsOf(
        type: CategoryType,
        self: UUID?,
    ): List<Category> = loaded.filter { it.type == type && it.parentId == null && it.id != self }

    private fun ready(): CategoriesUiState.Ready {
        val children = loaded.filter { it.parentId != null }.groupBy { it.parentId }
        val nodes = loaded
            .filter { it.parentId == null }
            .map { CategoryNode(it, children[it.id].orEmpty()) }
        return CategoriesUiState.Ready(
            income = nodes.filter { it.category.type == CategoryType.income },
            expense = nodes.filter { it.category.type == CategoryType.expense },
        )
    }
}

/** Правка поля гасит ошибку под ним: она была про прошлую попытку. */
private fun CategoryEditUiState.cleared(field: String): CategoryEditUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

// Детали 422 ложатся под поля; общий текст остаётся только для того, что под поле не легло.
private fun CategoryEditUiState.failed(failure: ApiFailure): CategoryEditUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underFields = details.filter { it.`field` in formFields }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        error = if (details.isNotEmpty() && underFields.size == details.size) null else failure.toUiError(),
    )
}
