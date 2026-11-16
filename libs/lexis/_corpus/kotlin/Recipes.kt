// Recipes.kt - scaling recipes and building a shopping list.
package io.example.kitchen

import java.time.Duration
import kotlin.math.ceil
import kotlin.math.roundToInt

const val DEFAULT_SERVINGS = 4
private const val GRAMS_PER_OUNCE = 28.3495

enum class Unit(val symbol: String, val metric: Boolean = true) {
    GRAM("g"),
    MILLILITRE("ml"),
    OUNCE("oz", metric = false),
    PIECE("");

    fun format(amount: Double): String =
        if (this == PIECE) amount.roundToInt().toString() else "%.1f%s".format(amount, symbol)
}

data class Ingredient(val name: String, val amount: Double, val unit: Unit = Unit.GRAM) {
    operator fun times(factor: Double) = copy(amount = amount * factor)

    fun toMetric(): Ingredient = when (unit) {
        Unit.OUNCE -> copy(amount = amount * GRAMS_PER_OUNCE, unit = Unit.GRAM)
        else -> this
    }
}

sealed interface Step {
    val text: String

    data class Prepare(override val text: String) : Step
    data class Cook(override val text: String, val time: Duration, val celsius: Int? = null) : Step
    data object Rest : Step {
        override val text = "Leave to rest."
    }
}

class Recipe(
    val title: String,
    val servings: Int = DEFAULT_SERVINGS,
    private val ingredients: List<Ingredient>,
    val steps: List<Step> = emptyList(),
) {
    init {
        require(servings > 0) { "a recipe serves at least one, not $servings" }
    }

    val totalTime: Duration
        get() = steps.filterIsInstance<Step.Cook>().fold(Duration.ZERO) { acc, s -> acc + s.time }

    /** The same recipe for a different number of people. */
    fun scaledTo(people: Int): Recipe {
        val factor = people.toDouble() / servings
        return Recipe(title, people, ingredients.map { it * factor }, steps)
    }

    fun shoppingList(): Map<String, Ingredient> =
        ingredients
            .map(Ingredient::toMetric)
            .groupBy { it.name.lowercase() }
            .mapValues { (_, same) -> same.reduce { a, b -> a.copy(amount = a.amount + b.amount) } }

    override fun toString() = "$title (serves $servings, ${totalTime.toMinutes()} min)"
}

@DslMarker
annotation class RecipeDsl

@RecipeDsl
class RecipeBuilder(private val title: String) {
    private val ingredients = mutableListOf<Ingredient>()
    private val steps = mutableListOf<Step>()
    var servings = DEFAULT_SERVINGS

    infix fun Double.of(name: String) = ingredients.add(Ingredient(name, this))
    fun pieces(count: Int, name: String) = ingredients.add(Ingredient(name, count.toDouble(), Unit.PIECE))
    fun prepare(text: String) { steps += Step.Prepare(text) }
    fun cook(minutes: Long, celsius: Int? = null, text: () -> String) {
        steps += Step.Cook(text(), Duration.ofMinutes(minutes), celsius)
    }

    fun build() = Recipe(title, servings, ingredients.toList(), steps.toList())
}

inline fun recipe(title: String, block: RecipeBuilder.() -> kotlin.Unit): Recipe =
    RecipeBuilder(title).apply(block).build()

fun <T : Comparable<T>> List<T>.median(): T? = sorted().getOrNull(size / 2)

fun describe(step: Step): String = when (step) {
    is Step.Prepare -> "- ${step.text}"
    is Step.Cook -> buildString {
        append("- ${step.text} for ${step.time.toMinutes()} min")
        step.celsius?.let { append(" at $it°C") }
    }
    Step.Rest -> "- ${step.text}"
}

fun main(args: Array<String>) {
    val bread = recipe("Soda bread") {
        servings = 6
        450.0 of "flour"
        8.5 of "salt"
        pieces(1, "egg")
        prepare("Mix the dry ingredients.")
        cook(minutes = 35, celsius = 200) { "Bake until it sounds hollow" }
    }

    val people = args.firstOrNull()?.toIntOrNull() ?: 9
    val scaled = bread.scaledTo(people)
    println(scaled)
    for ((name, item) in scaled.shoppingList()) {
        println("  ${name.padEnd(8)} ${item.unit.format(ceil(item.amount))}")
    }
    scaled.steps.plus(Step.Rest).map(::describe).forEach(::println)

    val raw = """
        |Loaves: ${people / bread.servings}
        |Note: "keeps two days"
    """.trimMargin()
    println(raw)
    val odd = listOf(3, 0x1F, 1_000, 7L.toInt()).median() ?: -1
    check(odd > 0 && 'x' != '\n') { "median was $odd" }
}
