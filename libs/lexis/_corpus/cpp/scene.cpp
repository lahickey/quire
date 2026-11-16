// scene.cpp - a small scene graph with typed components.
#include <algorithm>
#include <cassert>
#include <cstdint>
#include <iostream>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <unordered_map>
#include <vector>

#include "scene.hpp"

#define SCENE_VERSION 3

namespace scene {

using NodeId = std::uint32_t;
constexpr NodeId kRoot = 0;
constexpr double kEpsilon = 1e-9;

struct Vec3 {
    float x{}, y{}, z{};

    constexpr Vec3 operator+(const Vec3& o) const noexcept { return {x + o.x, y + o.y, z + o.z}; }
    constexpr Vec3 operator*(float s) const noexcept { return {x * s, y * s, z * s}; }
    bool operator==(const Vec3&) const = default;
};

enum class Flag : std::uint8_t { None = 0, Hidden = 1 << 0, Static = 1 << 1 };

template <typename T>
concept Component = requires(T t) {
    { T::name() } -> std::convertible_to<std::string_view>;
    { t.update(0.0f) } -> std::same_as<void>;
};

class Node final {
public:
    explicit Node(NodeId id, std::string name) : id_(id), name_(std::move(name)) {}
    Node(const Node&) = delete;
    Node& operator=(const Node&) = delete;
    virtual ~Node() = default;

    [[nodiscard]] NodeId id() const noexcept { return id_; }
    [[nodiscard]] const std::string& name() const noexcept { return name_; }

    Node& add_child(std::unique_ptr<Node> child) {
        assert(child != nullptr);
        child->parent_ = this;
        children_.push_back(std::move(child));
        return *children_.back();
    }

    /* Depth-first, parents before children. */
    template <typename Fn>
    void visit(Fn&& fn, int depth = 0) const {
        fn(*this, depth);
        for (const auto& c : children_) {
            c->visit(fn, depth + 1);
        }
    }

    Vec3 world_position() const {
        Vec3 p = position;
        for (const Node* n = parent_; n != nullptr; n = n->parent_) {
            p = p + n->position;
        }
        return p;
    }

    Vec3 position{};
    Flag flags = Flag::None;

private:
    NodeId id_;
    std::string name_;
    Node* parent_ = nullptr;
    std::vector<std::unique_ptr<Node>> children_;
};

struct Spinner {
    static constexpr std::string_view name() { return "spinner"; }
    void update(float dt) { angle += speed * dt; }
    float angle = 0.0f, speed = 1.5f;
};

template <Component C>
class Pool {
public:
    C& attach(NodeId id) { return items_.try_emplace(id).first->second; }

    std::optional<C> find(NodeId id) const {
        if (auto it = items_.find(id); it != items_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    void update_all(float dt) {
        std::for_each(items_.begin(), items_.end(), [dt](auto& kv) { kv.second.update(dt); });
    }

private:
    std::unordered_map<NodeId, C> items_;
};

std::ostream& operator<<(std::ostream& os, const Vec3& v) {
    return os << '(' << v.x << ", " << v.y << ", " << v.z << ')';
}

}  // namespace scene

int main() try {
    using namespace scene;

    Node root{kRoot, "root"};
    auto& arm = root.add_child(std::make_unique<Node>(1, "arm"));
    arm.position = {1.0f, 0.5f, 0.0f};
    auto& hand = arm.add_child(std::make_unique<Node>(2, R"(hand "left")"));
    hand.position = Vec3{0.25f, 0, 0} * 2.0f;

    Pool<Spinner> spinners;
    spinners.attach(hand.id()).speed = 3.0e-1f;
    spinners.update_all(1.0f / 60);

    std::size_t count = 0;
    root.visit([&count](const Node& n, int depth) {
        std::cout << std::string(static_cast<std::size_t>(depth) * 2, ' ') << n.name() << ' ' << n.world_position() << '\n';
        ++count;
    });

    const auto big = 1'000'000ULL;
    std::cout << count << " nodes, v" << SCENE_VERSION << ", " << big << "\n";
    return count == 3 ? 0 : 1;
} catch (const std::exception& e) {
    std::cerr << "error: " << e.what() << std::endl;
    throw;
}
