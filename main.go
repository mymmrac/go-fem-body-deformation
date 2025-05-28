package main

import (
	"log/slog"
	"strconv"

	gui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	camera  rl.Camera3D
	numbers []rl.Texture2D
)

type BodyDrawOptions struct {
	ShowEdges    bool
	ShowVertexes bool
}

func main() {
	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(1280, 720, "Body Deformation")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.SetWindowState(rl.FlagWindowResizable)

	camera = rl.NewCamera3D(
		rl.NewVector3(10, 10, 10),
		rl.NewVector3(0, 0, 0),
		rl.NewVector3(0, 1, 0),
		45,
		rl.CameraPerspective,
	)
	cameraOrbiting := false

	bodySize := [3]*InputValue[float64]{
		NewInputValue(4.0),
		NewInputValue(5.0),
		NewInputValue(3.0),
	}

	bodySplit := [3]*InputValue[int]{
		NewInputValue(4),
		NewInputValue(8),
		NewInputValue(3),
	}

	yungaModule := NewInputValue(4.0)
	poissonRatio := NewInputValue(0.3)
	pressure := NewInputValue(2.0)

	fem := &FEM{
		zu: make(map[ElementSide]bool),
		zp: make(map[ElementSide]bool),
	}
	body, bodyIndexes := fem.BuildElements(InputsToSlice3(bodySize), InputsToSlice3(bodySplit))
	var deformedBody [][3]float64

	{ // Fix bottom and push on top
		a, b, c := bodySplit[0].Value, bodySplit[1].Value, bodySplit[2].Value
		for i := range a * b {
			fem.zu[ElementSide{i, 4}] = true
			fem.zp[ElementSide{i + a*b*(c-1), 5}] = true
		}
	}

	// TODO: Remove this
	// fem.zp[ElementSide{92, 0}] = true
	// var rotation = rl.MatrixRotate(rl.GetCameraUp(&camera), 4.5)
	// var view = rl.Vector3Subtract(camera.Position, camera.Target)
	// view = rl.Vector3Transform(view, rotation)
	// camera.Position = rl.Vector3Add(camera.Target, view)
	// rl.CameraMoveToTarget(&camera, -7)

	const inputTextSize = 20
	var (
		inputWidth  float32 = 64.0 * 1.4
		inputHeight float32 = 24.0 * 1.4
		padding     float32 = 8.0 * 1.4
	)

	a0 := rl.NewVector3(0, 0, 0)
	aX := rl.NewVector3(1, 0, 0)
	aY := rl.NewVector3(0, 1, 0)
	aZ := rl.NewVector3(0, 0, 1)

	for i := range 1 << 11 {
		textImage := rl.ImageTextEx(rl.GetFontDefault(), strconv.Itoa(i), 32, 4, rl.White)
		numbers = append(numbers, rl.LoadTextureFromImage(textImage))
		rl.UnloadImage(textImage)
	}

	showNumbers := false
	showOriginal := true
	showForces := true
	opt := BodyDrawOptions{
		ShowEdges:    true,
		ShowVertexes: false,
	}

	running := 0

	quads := [6][6]int{
		0: {1, 3, 2, 1, 0, 3},
		1: {1, 3, 2, 1, 0, 3},

		2: {1, 3, 2, 1, 0, 3},
		3: {1, 3, 2, 1, 0, 3},

		4: {0, 1, 2, 3, 0, 2},
		5: {0, 2, 1, 3, 2, 0},
	}

	for !rl.WindowShouldClose() {
		topLeftUiRect := rl.NewRectangle(
			0, 0,
			padding+inputWidth*3.5+padding*2.5+padding,
			padding+inputHeight*2+padding+padding,
		)
		bottomLeftUiRect := rl.NewRectangle(
			0, float32(rl.GetScreenHeight())-(padding+inputHeight*3+padding*2+padding),
			padding+inputWidth*2+padding+padding,
			padding+inputHeight*3+padding*2+padding,
		)

		if rl.IsKeyPressed(rl.KeySpace) {
			cameraOrbiting = !cameraOrbiting
		}

		if rl.IsMouseButtonDown(rl.MouseButtonLeft) && (!rl.CheckCollisionPointRec(rl.GetMousePosition(), topLeftUiRect) &&
			!rl.CheckCollisionPointRec(rl.GetMousePosition(), bottomLeftUiRect)) {
			md := rl.GetMouseDelta()
			rl.CameraYaw(&camera, -md.X*0.003, 1)
			rl.CameraPitch(&camera, -md.Y*0.003, 1, 1, 0)
		} else if cameraOrbiting {
			var rotation = rl.MatrixRotate(rl.GetCameraUp(&camera), 0.5*rl.GetFrameTime())
			var view = rl.Vector3Subtract(camera.Position, camera.Target)
			view = rl.Vector3Transform(view, rotation)
			camera.Position = rl.Vector3Add(camera.Target, view)
		}

		rl.CameraMoveToTarget(&camera, -rl.GetMouseWheelMove())
		if rl.IsKeyPressed(rl.KeyKpSubtract) {
			rl.CameraMoveToTarget(&camera, 2.0)
		}
		if rl.IsKeyPressed(rl.KeyKpAdd) {
			rl.CameraMoveToTarget(&camera, -2.0)
		}

		const scaleFactor = 0.95
		if rl.IsKeyPressed(rl.KeyMinus) {
			inputWidth *= scaleFactor
			inputHeight *= scaleFactor
			padding *= scaleFactor
		}
		if rl.IsKeyPressed(rl.KeyEqual) {
			inputWidth /= scaleFactor
			inputHeight /= scaleFactor
			padding /= scaleFactor
		}

		if rl.IsKeyPressed(rl.KeyO) {
			showOriginal = !showOriginal
		}
		if showOriginal && rl.IsKeyPressed(rl.KeyN) {
			showNumbers = !showNumbers
		}
		if showOriginal && rl.IsKeyPressed(rl.KeyF) {
			showForces = !showForces
		}
		if rl.IsKeyPressed(rl.KeyE) {
			opt.ShowEdges = !opt.ShowEdges
		}
		if rl.IsKeyPressed(rl.KeyV) {
			opt.ShowVertexes = !opt.ShowVertexes
		}

		if showOriginal && showForces {
			if rl.IsKeyPressed(rl.KeyC) {
				if rl.IsKeyDown(rl.KeyLeftShift) {
					clear(fem.zu)
				} else {
					clear(fem.zp)
				}
			}

			if rl.IsKeyPressed(rl.KeyT) {
				a, b, c := bodySplit[0].Value, bodySplit[1].Value, bodySplit[2].Value
				fixOrPush := rl.IsKeyDown(rl.KeyLeftShift)
				for i := range a * b {
					es := ElementSide{i + a*b*(c-1), 5}
					if fixOrPush {
						fem.zu[es] = true
						fem.zp[es] = false
					} else {
						fem.zu[es] = false
						fem.zp[es] = true
					}
				}
			}

			if rl.IsKeyPressed(rl.KeyB) {
				a, b, _ := bodySplit[0].Value, bodySplit[1].Value, bodySplit[2].Value
				fixOrPush := rl.IsKeyDown(rl.KeyLeftShift)
				for i := range a * b {
					es := ElementSide{i, 4}
					if fixOrPush {
						fem.zu[es] = true
						fem.zp[es] = false
					} else {
						fem.zu[es] = false
						fem.zp[es] = true
					}
				}
			}
		}

		ray := rl.GetScreenToWorldRay(rl.GetMousePosition(), camera)
		rl.BeginDrawing()
		{
			rl.ClearBackground(rl.RayWhite)

			rl.BeginMode3D(camera)
			{
				rl.DrawGrid(32, 1)

				origin := rl.Vector3Scale(InputsToVec3(bodySize), 0.5)
				origin.Y, origin.Z = origin.Z, origin.Y
				origin.Y = 0

				if showOriginal {
					drawBody(body, bodyIndexes, origin, rl.Gray, rl.Blue, showNumbers, opt)

					if showForces {
						a, b, c := bodySplit[0].Value, bodySplit[1].Value, bodySplit[2].Value

						collisions := make(map[int]map[int]rl.RayCollision)
						for i, cube := range fem.elements {
							y := i / (b * a)
							z := (i % (b * a)) / a
							x := i % a

							if x != 0 && y != 0 && z != 0 && x != a-1 && y != c-1 && z != b-1 {
								continue
							}

							sides := make([]int, 0)

							if x == 0 {
								sides = append(sides, 0)
							}
							if x == a-1 {
								sides = append(sides, 1)
							}

							if z == 0 {
								sides = append(sides, 2)
							}
							if z == b-1 {
								sides = append(sides, 3)
							}

							if y == 0 {
								sides = append(sides, 4)
							}
							if y == c-1 {
								sides = append(sides, 5)
							}

							for _, n := range sides {
								side := fem.choseCubeSide(cube, n)
								collision := rl.GetRayCollisionQuad(ray,
									transformPoint(side[0], origin), transformPoint(side[1], origin),
									transformPoint(side[2], origin), transformPoint(side[3], origin),
								)
								if collision.Hit {
									if collisions[i] == nil {
										collisions[i] = make(map[int]rl.RayCollision)
									}
									collisions[i][n] = collision
								}
							}
						}

						closestCollisionI := -1
						closestCollisionN := -1
						for i, collisionN := range collisions {
							for n, collision := range collisionN {
								if closestCollisionI == -1 && closestCollisionN == -1 {
									closestCollisionI = i
									closestCollisionN = n
									continue
								}
								if collision.Distance < collisions[closestCollisionI][closestCollisionN].Distance {
									closestCollisionI = i
									closestCollisionN = n
								}
							}
						}

						for i, cube := range fem.elements {
							y := i / (b * a)
							z := (i % (b * a)) / a
							x := i % a

							if x != 0 && y != 0 && z != 0 && x != a-1 && y != c-1 && z != b-1 {
								continue
							}

							sides := make([]int, 0)

							if x == 0 {
								sides = append(sides, 0)
							}
							if x == a-1 {
								sides = append(sides, 1)
							}

							if z == 0 {
								sides = append(sides, 2)
							}
							if z == b-1 {
								sides = append(sides, 3)
							}

							if y == 0 {
								sides = append(sides, 4)
							}
							if y == c-1 {
								sides = append(sides, 5)
							}

							for n := range 6 {
								var chosen int // 0 - nothing, 1 - fix, 2 - push
								es := ElementSide{i, n}
								if fem.zu[es] {
									chosen = 1
								} else if fem.zp[es] {
									chosen = 2
								}

								if (closestCollisionI == i && closestCollisionN == n) || chosen != 0 {
									if (closestCollisionI == i && closestCollisionN == n) && rl.IsMouseButtonPressed(rl.MouseButtonRight) {
										if rl.IsKeyDown(rl.KeyLeftShift) {
											fem.zu[es] = !(chosen != 0)
											fem.zp[es] = false
										} else {
											fem.zu[es] = false
											fem.zp[es] = !(chosen != 0)
										}
									}

									q := quads[n]
									side := fem.choseCubeSide(cube, n)

									clr := rl.ColorAlpha(rl.LightGray, 0.7)

									if chosen == 1 {
										clr = rl.ColorAlpha(rl.Blue, 0.4)
										if collisions[i] != nil && collisions[i][n].Hit {
											clr = rl.ColorAlpha(rl.Blue, 0.7)
										}
									} else if chosen == 2 {
										clr = rl.ColorAlpha(rl.Orange, 0.4)
										if collisions[i] != nil && collisions[i][n].Hit {
											clr = rl.ColorAlpha(rl.Orange, 0.7)
										}
									}

									// rl.DrawBillboard(camera, numbers[0], rl.Vector3Add(transformPoint(side[0], origin), rl.Vector3{Y: 0.2}), 0.2, rl.Black)
									// rl.DrawBillboard(camera, numbers[1], rl.Vector3Add(transformPoint(side[1], origin), rl.Vector3{Y: 0.2}), 0.2, rl.Black)
									// rl.DrawBillboard(camera, numbers[2], rl.Vector3Add(transformPoint(side[2], origin), rl.Vector3{Y: 0.2}), 0.2, rl.Black)
									// rl.DrawBillboard(camera, numbers[3], rl.Vector3Add(transformPoint(side[3], origin), rl.Vector3{Y: 0.2}), 0.2, rl.Black)

									rl.DrawTriangle3D(transformPoint(side[q[0]], origin), transformPoint(side[q[1]], origin), transformPoint(side[q[2]], origin), clr)
									rl.DrawTriangle3D(transformPoint(side[q[3]], origin), transformPoint(side[q[4]], origin), transformPoint(side[q[5]], origin), clr)
								}
							}
						}
					}
				}
				if deformedBody != nil {
					drawBody(deformedBody, bodyIndexes, origin, rl.Red, rl.Green, false, opt)
				}

				const thickness = 0.02
				rl.DrawCylinderEx(a0, aX, thickness, thickness, 8, rl.Red)
				rl.DrawCylinderEx(a0, aY, thickness, thickness, 8, rl.Green)
				rl.DrawCylinderEx(a0, aZ, thickness, thickness, 8, rl.Blue)

				// TODO: Remove?
				// for i, point := range localPoints3D {
				// 	p := transformPoint(point, rl.Vector3{})
				// 	rl.DrawSphere(p, 0.05, rl.Red)
				// 	rl.DrawBillboard(camera, numbers[i], rl.Vector3Add(p, rl.Vector3{Y: 0.2}), 0.2, rl.Black)
				// }
			}
			rl.EndMode3D()

			rl.DrawRectangleRec(topLeftUiRect, rl.RayWhite)
			rl.DrawRectangleLinesEx(topLeftUiRect, 1, rl.Gray)

			rl.DrawRectangleRec(bottomLeftUiRect, rl.RayWhite)
			rl.DrawRectangleLinesEx(bottomLeftUiRect, 1, rl.Gray)

			bodyUpdated := false

			// Sizes
			gui.Label(rl.NewRectangle(padding, padding, inputWidth/2, inputHeight), "Size")
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*(float32(i)+0.5), padding, inputWidth, inputHeight),
					&bodySize[i].Text, inputTextSize, bodySize[i].Edit,
				) {
					bodySize[i].ToggleEdit()
					v, err := strconv.ParseFloat(bodySize[i].Text, 32)
					if err != nil {
						slog.Error("Invalid size value", "err", err)
					} else {
						v = max(min(v, 1000), 0.01)
						bodySize[i].Value = v
						bodyUpdated = true
					}
					bodySize[i].UpdateText()
				}
			}

			// Splits
			gui.Label(rl.NewRectangle(padding, padding+padding+inputHeight, inputWidth/2, inputHeight), "Split")
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*(float32(i)+0.5), padding+inputHeight+padding, inputWidth, inputHeight),
					&bodySplit[i].Text, inputTextSize, bodySplit[i].Edit,
				) {
					bodySplit[i].ToggleEdit()
					v, err := strconv.ParseInt(bodySplit[i].Text, 10, 32)
					if err != nil {
						slog.Error("Invalid split value", "err", err, "value", v)
					} else {
						v = max(min(v, 100), 1)
						bodySplit[i].Value = int(v)
						bodyUpdated = true
					}
					bodySplit[i].UpdateText()
				}
			}

			// Young's modulus
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight), "Young's Modulus")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight),
				&yungaModule.Text, inputTextSize, yungaModule.Edit,
			) {
				yungaModule.ToggleEdit()
				v, err := strconv.ParseFloat(yungaModule.Text, 64)
				if err != nil {
					slog.Error("Invalid Young's modulus value", "err", err)
				} else {
					yungaModule.Value = max(min(v, 100000.0), 0.01)
				}
				yungaModule.UpdateText()
			}

			// Poisson's ratio
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight), "Poisson's ratio")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight),
				&poissonRatio.Text, inputTextSize, poissonRatio.Edit,
			) {
				poissonRatio.ToggleEdit()
				v, err := strconv.ParseFloat(poissonRatio.Text, 64)
				if err != nil {
					slog.Error("Invalid Poisson's ratio value", "err", err)
				} else {
					poissonRatio.Value = max(min(v, 0.49), 0.0)
				}
				poissonRatio.UpdateText()
			}

			// Pressure
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight), "Pressure")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight),
				&pressure.Text, inputTextSize, pressure.Edit,
			) {
				pressure.ToggleEdit()
				v, err := strconv.ParseFloat(pressure.Text, 64)
				if err != nil {
					slog.Error("Invalid pressure value", "err", err)
				} else {
					pressure.Value = max(min(v, 10000.0), 0.01)
				}
				pressure.UpdateText()
			}

			if bodyUpdated {
				body, bodyIndexes = fem.BuildElements(InputsToSlice3(bodySize), InputsToSlice3(bodySplit))
				deformedBody = nil
			}

			// Run
			runBtnText := "Run"
			if running > 0 {
				runBtnText = "Running..."
				running++
			}
			if gui.Button(
				rl.NewRectangle(float32(rl.GetScreenWidth())-padding-inputWidth, float32(rl.GetScreenHeight())-padding-inputHeight, inputWidth, inputHeight),
				runBtnText,
			) || rl.IsKeyPressed(rl.KeyEnter) {
				running = 1
			} else if running > 10 {
				slog.Info("Running...",
					"bodySize", InputsToVec3(bodySize),
					"bodySplits", InputsToVec3(bodySplit),
					"yungaModule", yungaModule, "poissonRatio", poissonRatio, "pressure", pressure,
				)
				deformedBody = fem.ApplyForce(yungaModule.Value, poissonRatio.Value, pressure.Value)
				running = 0
			}
		}
		rl.EndDrawing()
	}
}

func transformPoint(p [3]float64, origin rl.Vector3) rl.Vector3 {
	return rl.Vector3Subtract(rl.NewVector3(float32(p[0]), float32(p[2]), float32(p[1])), origin)
}

func drawBody(
	body [][3]float64, bodyIndexes map[[3]int]int, origin rl.Vector3,
	edgesColor, verticesColor rl.Color, showNumbers bool, opt BodyDrawOptions,
) {
	for key, idx := range bodyIndexes {
		p1 := transformPoint(body[idx], origin)

		const vertexSize = 0.07
		if opt.ShowVertexes {
			rl.DrawCube(p1, vertexSize, vertexSize, vertexSize, verticesColor)
		}

		if opt.ShowEdges {
			i, j, k := key[0], key[1], key[2]
			for _, delta := range [][3]int{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} {
				ni, nj, nk := i+delta[0], j+delta[1], k+delta[2]
				neighborKey := [3]int{ni, nj, nk}
				if nIdx, ok := bodyIndexes[neighborKey]; ok {
					p2 := transformPoint(body[nIdx], origin)
					rl.DrawLine3D(p1, p2, edgesColor)
				}
			}
		}

		if showNumbers {
			rl.DrawBillboard(camera, numbers[idx+1], rl.Vector3Add(p1, rl.Vector3{Y: 0.2}), 0.2, rl.Black)
		}
	}
}
